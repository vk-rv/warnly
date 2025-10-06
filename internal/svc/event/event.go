// Package event provides the implementation of the event service.
// It includes methods for ingesting events from other services.
package event

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/vk-rv/warnly/internal/warnly"

	"golang.org/x/sync/singleflight"
)

// EventService provides event operations.
type EventService struct {
	projectStore warnly.ProjectStore
	issueStore   warnly.IssueStore
	cache        *cache.Cache
	sf           *singleflight.Group
	olap         warnly.AnalyticsStore
	now          func() time.Time
}

// NewEventService is a constructor of event service.
func NewEventService(
	projectStore warnly.ProjectStore,
	issueStore warnly.IssueStore,
	inMemCache *cache.Cache,
	olap warnly.AnalyticsStore,
	now func() time.Time,
) *EventService {
	return &EventService{
		projectStore: projectStore,
		issueStore:   issueStore,
		cache:        inMemCache,
		olap:         olap,
		sf:           &singleflight.Group{},
		now:          now,
	}
}

// IngestEvent ingests a new event into the system.
func (s *EventService) IngestEvent(ctx context.Context, req warnly.IngestRequest) (warnly.IngestEventResult, error) {
	res := warnly.IngestEventResult{}

	opts, err := s.getProjectOptions(ctx, req)
	if err != nil {
		return res, err
	}

	ipv4, ipv6, err := s.extractIP(req.IP)
	if err != nil {
		return res, err
	}

	event := req.Event

	eventHash, err := warnly.GetNormalizedHash(event)
	if err != nil {
		return res, err
	}

	cacheKey := fmt.Sprintf("%d:%s", req.ProjectID, eventHash)

	exceptionType := warnly.GetExceptionType(event.Exception, event.Message)
	exceptionValue := warnly.GetExceptionValue(event.Exception, warnly.DefaultMessage)

	issueInfo := warnly.IssueInfo{}
	var ok bool
	iss, found := s.cache.Get(cacheKey)
	if found {
		issueInfo, ok = iss.(warnly.IssueInfo)
		if !ok {
			return res, errors.New("event service ingest: cache issue info type assertion")
		}
		_, err, _ := s.sf.Do(cacheKey, s.updateLastSeen(ctx, issueInfo.ID, s.now().UTC()))
		if err != nil {
			return res, fmt.Errorf("event service ingest: update last seen %w", err)
		}
	} else {
		issue, err := s.issueStore.GetIssue(ctx, warnly.GetIssueCriteria{
			ProjectID: req.ProjectID,
			Hash:      eventHash,
		})
		if err != nil {
			if !errors.Is(err, warnly.ErrNotFound) {
				return res, fmt.Errorf("event service ingest: get issue from store %w", err)
			}
			now := s.now().UTC()
			issue = &warnly.Issue{
				ID:          0, // set by store
				UUID:        warnly.NewUUID(),
				FirstSeen:   now,
				LastSeen:    now,
				Hash:        eventHash,
				Message:     exceptionValue,
				ErrorType:   exceptionType,
				View:        warnly.GetBreaker(event.Exception),
				NumComments: 0,
				ProjectID:   req.ProjectID,
				Priority:    warnly.PriorityHigh,
			}
			if _, err, _ := s.sf.Do(cacheKey, s.storeIssue(ctx, issue)); err != nil {
				return res, fmt.Errorf("event service ingest: store issue %w", err)
			}
		} else {
			if _, err, _ := s.sf.Do(cacheKey, s.updateLastSeen(ctx, issueInfo.ID, s.now().UTC())); err != nil {
				return res, fmt.Errorf("event service ingest: update last seen %w", err)
			}
		}
		issueInfo = warnly.IssueInfo{ID: issue.ID, UUID: issue.UUID.String(), Hash: issue.Hash}
		s.cache.Set(cacheKey, issueInfo, cache.DefaultExpiration)
	}

	tkv := makeTags(event)

	ev := &warnly.EventClickhouse{
		EventID:       event.EventID,
		Deleted:       0,
		GroupID:       uint64(issueInfo.ID),
		RetentionDays: opts.RetentionDays,
		User:          makeUser(event),
		UserEmail:     event.User.Email,
		UserName:      event.User.Name,
		UserUsername:  event.User.Username,
		ProjectID:     uint16(req.ProjectID),
		Type:          warnly.EventTypeException,
		CreatedAt:     event.Timestamp.UTC(),
		Platform:      uint8(warnly.PlatformByName(event.Platform)),
		Env:           event.Environment,
		Release:       event.Release,
		Message:       event.Message,
		Level:         warnly.GetLevel(event.Level),
		SDKID:         warnly.GetSDKID(event.SDK.Name),
		SDKVersion:    event.SDK.Version,
		Title:         exceptionType + ": " + exceptionValue,
		IPv4:          ipv4,
		IPv6:          ipv6,
		ContextsKey: []string{
			"device.arch",
			"device.num_cpu",
			"os.name",
			"trace.span_id",
			"trace.trace_id",
		},
		ContextsValue: []string{
			event.Contexts.Device.Arch,
			strconv.Itoa(event.Contexts.Device.NumCPU),
			event.Contexts.OS.Name,
			event.Contexts.Trace.SpanID,
			event.Contexts.Trace.TraceID,
		},
		TagsKey:                 tkv.keys,
		TagsValue:               tkv.values,
		PrimaryHash:             issueInfo.UUID,
		ExceptionStacksType:     warnly.GetExceptionStackTypes(event.Exception),
		ExceptionStacksValue:    warnly.GetExceptionStackValues(event.Exception),
		ExceptionFramesAbsPath:  warnly.GetExceptionFramesAbsPath(event.Exception),
		ExceptionFramesColNo:    warnly.GetExceptionFramesColNo(event.Exception),
		ExceptionFramesFilename: warnly.GetExceptionFramesFilename(event.Exception),
		ExceptionFramesFunction: warnly.GetExceptionFramesFunction(event.Exception),
		ExceptionFramesLineNo:   warnly.GetExceptionFramesLineNo(event.Exception),
		ExceptionFramesInApp:    warnly.GetExceptionFramesInApp(event.Exception),
	}

	if err := s.olap.StoreEvent(ctx, ev); err != nil {
		return res, fmt.Errorf("event service ingest: store event in olap %w", err)
	}

	res.EventID = ev.EventID

	return res, nil
}

type tagsKeyValue struct {
	keys   []string
	values []string
}

func makeTags(event *warnly.EventBody) tagsKeyValue {
	tagsKeys := []string{}
	tagsValues := []string{}

	if event.Environment != "" {
		tagsKeys = append(tagsKeys, "env")
		tagsValues = append(tagsValues, event.Environment)
	}
	if event.Level != "" {
		tagsKeys = append(tagsKeys, "level")
		tagsValues = append(tagsValues, event.Level)
	}
	if event.Release != "" {
		tagsKeys = append(tagsKeys, "release")
		tagsValues = append(tagsValues, event.Release)
	}
	if event.ServerName != "" {
		tagsKeys = append(tagsKeys, "server_name")
		tagsValues = append(tagsValues, event.ServerName)
	}
	if event.User.ID != "" {
		tagsKeys = append(tagsKeys, "user")
		tagsValues = append(tagsValues, "id:"+event.User.ID)
	}

	for k, v := range event.Tags {
		if k != "" && v != "" {
			tagsKeys = append(tagsKeys, k)
			tagsValues = append(tagsValues, v)
		}
	}

	return tagsKeyValue{keys: tagsKeys, values: tagsValues}
}

func makeUser(event *warnly.EventBody) string {
	user := ""
	if event.User.ID != "" {
		user = "id:" + event.User.ID
	}
	return user
}

// updateLastSeen updates the last seen time of an issue in oltp database.
func (s *EventService) updateLastSeen(ctx context.Context, issueID int64, now time.Time) func() (any, error) {
	return func() (any, error) {
		if err := s.issueStore.UpdateLastSeen(ctx, issueID, now); err != nil {
			return false, fmt.Errorf("event service update last seen: %w", err)
		}
		return true, nil
	}
}

// storeIssue stores an issue in oltp database.
func (s *EventService) storeIssue(ctx context.Context, issue *warnly.Issue) func() (any, error) {
	return func() (any, error) {
		if err := s.issueStore.StoreIssue(ctx, issue); err != nil {
			return false, fmt.Errorf("event service store issue: %w", err)
		}
		return true, nil
	}
}

// getProjectOptions retrieves project options such as event retention days from database.
func (s *EventService) getProjectOptions(ctx context.Context, req warnly.IngestRequest) (*warnly.ProjectOptions, error) {
	cacheKey := fmt.Sprintf("project_options:%d", req.ProjectID)
	if opts, found := s.cache.Get(cacheKey); found {
		projOpts, ok := opts.(*warnly.ProjectOptions)
		if !ok {
			return nil, errors.New("event service get project options: cache project options type assertion")
		}
		return projOpts, nil
	}

	opts, err := s.projectStore.GetOptions(ctx, req.ProjectID, req.ProjectKey)
	if err != nil {
		return nil, err
	}
	opts.RetentionDays = 90

	s.cache.Set(cacheKey, opts, time.Minute*10)

	return opts, nil
}

// extractIP extracts IPv4 and IPv6 addresses from a given IP address string.
func (s *EventService) extractIP(ipaddr string) (ipv4, ipv6 string, err error) {
	ip, _, err := net.SplitHostPort(ipaddr)
	if err != nil {
		return "", "", fmt.Errorf("ingest event: split host port: %w", err)
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "", "", fmt.Errorf("ingest event: remote addr is not an ip address: %w", err)
	}

	ipv4 = "127.0.0.1"
	ipv6 = "::1"
	if addr.Is4() {
		ipv4 = addr.String()
	} else {
		ipv6 = addr.String()
	}

	return ipv4, ipv6, nil
}
