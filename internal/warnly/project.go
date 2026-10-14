package warnly

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

const PageSize = 5

const (
	// PlatformGolang represents the Go platform.
	PlatformGolang Platform = iota + 1
	// PlatformRust represents the Rust platform.
	PlatformRust
)

// ErrProjectNotFound is an error that is returned when the project is not found.
var ErrProjectNotFound = errors.New("project not found")

// Project is a representation of a project in the system.
type Project struct {
	CreatedAt       time.Time      `json:"created_at"`
	Name            string         `json:"name"`
	Key             string         `json:"key"`
	Events          EventsList     `json:"events"`
	GroupMetrics    []GroupMetrics `json:"group_metrics"`
	IssueList       []IssueEntry   `json:"issue_list"`
	NewIssueList    []IssueEntry   `json:"new_issue_list"`
	ResultIssueList []IssueEntry   `json:"result_issue_list"`
	ID              int            `json:"id"`
	UserID          int            `json:"user_id"`
	TeamID          int            `json:"team_id"`
	AllLength       int            `json:"all_length"`
	NewLength       int            `json:"new_length"`
	Platform        Platform       `json:"platform"`
}

// IssueEntry is how we represent an issue in the system.
type IssueEntry struct {
	LastSeen      time.Time `json:"last_seen"`
	FirstSeen     time.Time `json:"first_seen"`
	Type          string    `json:"type"`
	View          string    `json:"view"`
	Message       string    `json:"message"`
	ID            int64     `json:"id"`
	TimesSeen     uint64    `json:"times_seen"`
	UserCount     uint64    `json:"user_count"`
	ProjectID     int       `json:"project_id"`
	MessagesCount int       `json:"messages_count"`
}

// TimeAgo returns a human-readable string representing the time since the issue was last seen.
func TimeAgo(now func() time.Time, t time.Time, narrow bool) string {
	duration := now().UTC().Sub(t.UTC())
	seconds := int(duration.Seconds())
	minutes := int(duration.Minutes())
	hours := int(duration.Hours())
	days := hours / 24
	months := days / 30
	years := days / 365

	switch {
	case duration < time.Minute:
		return formatTime(seconds, "sec", "seconds", narrow)
	case duration < time.Hour:
		return formatTime(minutes, "min", "minutes", narrow)
	case duration < time.Hour*24:
		return formatTime(hours, "h", "hours", narrow)
	case duration < time.Hour*24*30:
		return formatTime(days, "d", "days", narrow)
	case duration < time.Hour*24*365:
		return formatTime(months, "mo", "months", narrow)
	default:
		return formatTime(years, "y", "years", narrow)
	}
}

func formatTime(value int, narrowUnit, fullUnit string, narrow bool) string {
	if narrow {
		return fmt.Sprintf("%d%s", value, narrowUnit)
	}
	unit := fullUnit
	if value == 1 {
		unit = fullUnit[:len(fullUnit)-1]
	}
	return fmt.Sprintf("%d %s", value, unit)
}

type EventsList []EventsPerHour

type ListProjectsResult struct {
	Criteria *ListProjectsCriteria `json:"criteria"`
	Projects []Project             `json:"projects"`
	Teams    []Team                `json:"teams"`
}

// ProjectStore encapsulates the project storage.
type ProjectStore interface {
	// CreateProject creates a new project.
	CreateProject(ctx context.Context, project *Project) error
	// ListProjects returns a list of projects for the given teams.
	ListProjects(ctx context.Context, teamIDs []int, name string) ([]Project, error)
	// DeleteProject deletes a project by ID.
	DeleteProject(ctx context.Context, projectID int) error
	// GetProject returns a project by identifier.
	GetProject(ctx context.Context, projectID int) (*Project, error)
	// GetOptions returns the project options.
	GetOptions(ctx context.Context, projectID int, projectKey string) (*ProjectOptions, error)
}

type ProjectOptions struct {
	Name          string
	ID            int
	Platform      Platform
	RetentionDays uint8
}

// ProjectService encapsulates service domain logic.
//
//nolint:interfacebloat // think about how to refactor this
type ProjectService interface {
	// CreateProject creates a new project.
	CreateProject(ctx context.Context, req *CreateProjectRequest, user *User) (*ProjectInfo, error)
	// ListProjects returns a list of projects for the given teams.
	ListProjects(ctx context.Context, criteria *ListProjectsCriteria, user *User) (*ListProjectsResult, error)
	// ListTeams returns a list of teams associated with the user.
	ListTeams(ctx context.Context, user *User) ([]Team, error)
	// GetProject returns a project by identifier.
	GetProject(ctx context.Context, projectID int, user *User) (*Project, error)
	// DeleteProject deletes a project by ID.
	DeleteProject(ctx context.Context, projectID int, user *User) error
	// GetProjectDetails returns the project details.
	GetProjectDetails(ctx context.Context, req *ProjectDetailsRequest, user *User) (*ProjectDetails, error)
	// GetIssue returns the issue by ID.
	GetIssue(ctx context.Context, req *GetIssueRequest) (*IssueDetails, error)

	GetDiscussion(ctx context.Context, req *GetDiscussionsRequest) (*Discussion, error)

	// ListFields returns a list of fields related to an issue.
	// e.g. how many times a field like browser or os was seen in events.
	ListFields(ctx context.Context, req *ListFieldsRequest) (*ListFieldsResult, error)

	// ListEvents handles "All Errors" page per issue listing all error events.
	ListEvents(ctx context.Context, req *ListEventsRequest) (*ListEventsResult, error)

	// ListIssues returns a list of issues for specified projects.
	ListIssues(ctx context.Context, req *ListIssuesRequest) (*ListIssuesResult, error)

	// ListTeammates returns a list of teammates for the specified project.
	ListTeammates(ctx context.Context, req *ListTeammatesRequest) ([]Teammate, error)

	// CreateMessage creates a new message in the discussion.
	CreateMessage(ctx context.Context, req *CreateMessageRequest) error

	// DeleteMessage deletes a message in the discussion.
	DeleteMessage(ctx context.Context, req *DeleteMessageRequest) error

	// AssignIssue assigns an issue to a user.
	AssignIssue(ctx context.Context, req *AssignIssueRequest) error

	// DeleteAssignment unassigns an issue from a user.
	DeleteAssignment(ctx context.Context, req *UnassignIssueRequest) error

	// SearchProject searches for projects by name. Returns ErrProjectNotFound if no project is found.
	SearchProject(ctx context.Context, name string, user *User) (*Project, error)
	// ListPopularTags lists popular tag keys for search suggestions.
	ListPopularTags(ctx context.Context, req *ListPopularTagsRequest) ([]TagCount, error)
	// ListTagValues lists popular values for a given tag.
	ListTagValues(ctx context.Context, req *ListTagValuesRequest) ([]TagValueCount, error)
}

type DeleteMessageRequest struct {
	User      *User
	MessageID int
	ProjectID int
	IssueID   int
}

type ListPopularTagsRequest struct {
	User        *User
	ProjectName string
	Period      string
	Limit       int
}

type ListTagValuesRequest struct {
	User        *User
	Tag         string
	ProjectName string
	Period      string
	Start       string
	End         string
	Limit       int
}

// ListTeammatesRequest is a request to list teammates for a project.
type ListTeammatesRequest struct {
	User      *User
	ProjectID int
}

// Teammate is a user while listing teammates.
type Teammate struct {
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
	Username string `json:"username"`
	ID       int64  `json:"id"`
}

// AvatarInitials returns the initials of the teammate in uppercase.
func (t *Teammate) AvatarInitials() string {
	return strings.ToUpper(string(t.Name[0]) + string(t.Surname[0]))
}

// FullName returns the full name of the teammate.
func (t *Teammate) FullName() string {
	return t.Name + " " + t.Surname
}

type TeammateAssign struct {
	Teammate

	Assigned bool
}

type ListIssuesRequest struct {
	User        *User  `json:"user"`
	Period      string `json:"period"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Query       string `json:"query"`
	ProjectName string `json:"project_name"`
	ProjectIDs  []int  `json:"project_i_ds"`
	Offset      int    `json:"offset"`
	Limit       int    `json:"limit"`
}

type ListIssuesResult struct {
	RequestedProject string             `json:"requested_project"`
	Request          *ListIssuesRequest `json:"request"`
	LastProject      *Project           `json:"last_project"`
	Issues           []IssueEntry       `json:"issues"`
	Projects         []Project          `json:"projects"`
	PopularTags      []TagCount         `json:"popular_tags"`
	TotalIssues      int                `json:"total_issues"`
}

type GetAssignedFiltersCriteria struct {
	CurrentUserTeamIDs []int
}

type Filter struct {
	Key      string
	Operator string
	Value    string
}

//nolint:tagliatelle // support frontend format
type QueryToken struct {
	Key       string `json:"key"`
	Operator  string `json:"operator"`
	Value     string `json:"value"`
	IsRawText bool   `json:"isRawText"`
}

// ParseQuery parses a query string into QueryTokens.
// Supports quoted values, operators like : and !:, and raw text.
func ParseQuery(query string) []QueryToken {
	var tokens []QueryToken
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := range len(query) {
		char := query[i]

		switch {
		case !inQuotes && (char == '"' || char == '\''):
			inQuotes = true
			quoteChar = char
		case inQuotes && char == quoteChar:
			inQuotes = false
			quoteChar = 0
		case !inQuotes && char == ' ':
			if current.Len() > 0 {
				tokens = append(tokens, parseToken(current.String()))
				current.Reset()
			}
		default:
			current.WriteByte(char)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, parseToken(current.String()))
	}

	return tokens
}

// parseToken parses a single token string into QueryToken.
func parseToken(token string) QueryToken {
	if strings.Contains(token, ":") {
		parts := strings.SplitN(token, ":", 2)
		key := parts[0]
		value := parts[1]

		var operator string
		if strings.HasPrefix(value, "!") {
			operator = "is not"
			value = value[1:]
		} else {
			operator = "is"
		}

		// Remove quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		return QueryToken{
			Key:      key,
			Operator: operator,
			Value:    value,
		}
	}

	// Raw text, remove quotes if present
	value := token
	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
		value = value[1 : len(value)-1]
	}

	return QueryToken{
		Value:     value,
		IsRawText: true,
	}
}

type TagValueCount struct {
	Value string `json:"value"`
	Count uint64 `json:"count"`
}

type ListPopularTagsCriteria struct {
	From       time.Time
	To         time.Time
	ProjectIDs []int
	Limit      int
}

type ListTagValuesCriteria struct {
	From       time.Time
	To         time.Time
	Tag        string
	ProjectIDs []int
	Limit      int
}

func (l *ListIssuesResult) NoIssues() bool {
	return len(l.Issues) == 0
}

// ListEventsRequest is a request structure for listing all events per issue.
type ListEventsRequest struct {
	User      *User  `json:"user"`
	Period    string `json:"period"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Query     string `json:"query"`
	ProjectID int    `json:"project_id"`
	IssueID   int    `json:"issue_id"`
	Offset    int    `json:"offset"`
}

type ListEventsResult struct {
	Request     *ListEventsRequest `json:"request"`
	Events      []EventEntry       `json:"events"`
	PopularTags []TagCount         `json:"popular_tags"`
	ProjectID   int                `json:"project_id"`
	IssueID     int                `json:"issue_id"`
	TotalEvents uint64             `json:"total_events"`
	Offset      int                `json:"offset"`
}

type ListFieldsRequest struct {
	User      *User
	ProjectID int
	IssueID   int
}

type ListFieldsResult struct {
	ProjectName   string          `json:"project_name"`
	TagCount      []TagCount      `json:"tag_count"`
	FieldValueNum []FieldValueNum `json:"field_value_num"`
}

type Field struct{}

type GetDiscussionsRequest struct {
	User      *User
	ProjectID int
	IssueID   int
}

type CreateMessageRequest struct {
	User           *User
	Content        string
	MentionedUsers []int
	ProjectID      int
	IssueID        int
}

type Discussion struct {
	Info      DiscussionInfo `json:"info"`
	Teammates []Teammate     `json:"teammates"`
	Messages  []IssueMessage `json:"messages"`
}

type DiscussionInfo struct {
	IssueFirstSeen time.Time `json:"issue_first_seen"`
	ProjectID      int       `json:"project_id"`
	IssueID        int       `json:"issue_id"`
}

type GetIssueRequestSource string

const (
	GetIssueRequestSourceIssue GetIssueRequestSource = "issue"
)

type GetIssueRequest struct {
	User      *User                 `json:"user"`
	Period    string                `json:"period"`
	EventID   string                `json:"event_id"`
	Source    GetIssueRequestSource `json:"source"`
	ProjectID int                   `json:"project_id"`
	IssueID   int                   `json:"issue_id"`
}

type IssueEvent struct {
	UserID                  string    `json:"user_id"`
	UserEmail               string    `json:"user_email"`
	UserName                string    `json:"user_name"`
	UserUsername            string    `json:"user_username"`
	EventID                 string    `json:"event_id"`
	CreatedAt               time.Time `json:"created_at"`
	Env                     string    `json:"env"`
	Release                 string    `json:"release"`
	TagsKey                 []string  `json:"tags_key"`
	TagsValue               []string  `json:"tags_value"`
	ContextsKey             []string  `json:"contexts_key"`
	ContextsValue           []string  `json:"contexts_value"`
	Message                 string    `json:"message"`
	ExceptionFramesAbsPath  []string  `json:"exception_frames_abs_path"`
	ExceptionFramesColno    []int     `json:"exception_frames_colno"`
	ExceptionFramesFunction []string  `json:"exception_frames_function"`
	ExceptionFramesLineno   []int     `json:"exception_frames_lineno"`
	ExceptionFramesInApp    []int     `json:"exception_frames_in_app"`
}

type IssueDetails struct {
	LastSeen      time.Time        `json:"last_seen"`
	FirstSeen     time.Time        `json:"first_seen"`
	LastEvent     *IssueEvent      `json:"last_event"`
	Assignments   *Assignments     `json:"assignments"`
	Request       *GetIssueRequest `json:"request"`
	NextEventID   *string          `json:"next_event_id"`
	PrevEventID   *string          `json:"prev_event_id"`
	FirstEventID  *string          `json:"first_event_id"`
	LastEventID   *string          `json:"last_event_id"`
	View          string           `json:"view"`
	ErrorValue    string           `json:"error_value"`
	Message       string           `json:"message"`
	ErrorType     string           `json:"error_type"`
	ProjectName   string           `json:"project_name"`
	TagCount      []TagCount       `json:"tag_count"`
	TagValueNum   []FieldValueNum  `json:"tag_value_num"`
	Teammates     []Teammate       `json:"teammates"`
	StackDetails  []StackDetail    `json:"stack_details"`
	ProjectID     int              `json:"project_id"`
	Total24Hours  uint64           `json:"total24_hours"`
	Priority      IssuePriority    `json:"priority"`
	Total30Days   uint64           `json:"total30_days"`
	IssueID       int64            `json:"issue_id"`
	MessagesCount int              `json:"messages_count"`
	TimesSeen     uint64           `json:"times_seen"`
	UserCount     uint64           `json:"user_count"`
	IsNew         bool             `json:"is_new"`
	Platform      Platform         `json:"platform"`
}

func (id *IssueDetails) GetPlatform() string {
	return id.Platform.String()
}

func Cut(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

type TagKeyValue struct {
	Key   string
	Value string
}

func (id *IssueDetails) TagKeyValue() []TagKeyValue {
	if len(id.LastEvent.TagsKey) == 0 {
		return nil
	}

	res := make([]TagKeyValue, 0, len(id.LastEvent.TagsKey))
	for i := range id.LastEvent.TagsValue {
		res = append(res, TagKeyValue{
			Key:   id.LastEvent.TagsKey[i],
			Value: id.LastEvent.TagsValue[i],
		})
	}

	return res
}

func (id *IssueDetails) Contexts() map[string]string {
	if len(id.LastEvent.ContextsKey) == 0 {
		return nil
	}

	res := make(map[string]string, len(id.LastEvent.ContextsKey))
	for i := range id.LastEvent.ContextsKey {
		res[id.LastEvent.ContextsKey[i]] = id.LastEvent.ContextsValue[i]
	}
	return res
}

func (id *IssueDetails) Tag(tag string) string {
	idx := -1
	for i := range id.LastEvent.TagsKey {
		if id.LastEvent.TagsKey[i] == tag {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ""
	}
	return id.LastEvent.TagsValue[idx]
}

func (id *IssueDetails) ProgressLen(val string) string {
	for i := range id.TagValueNum {
		if id.TagValueNum[i].Value == val {
			percent := id.TagValueNum[i].PercentsOfTotal
			switch {
			case percent >= 100:
				return "w-full"
			case percent >= 75:
				return "w-3/4"
			case percent >= 50:
				return "w-1/2"
			case percent >= 25:
				return "w-1/4"
			default:
				return "w-1/5"
			}
		}
	}
	return ""
}

func (id *IssueDetails) EventID() string {
	return id.LastEvent.EventID
}

type StackDetail struct {
	Filepath     string `json:"filepath"`
	FunctionName string `json:"function_name"`
	LineNo       int    `json:"line_no"`
	InApp        bool   `json:"in_app"`
}

func (sd *StackDetail) InAppStr() string {
	if sd.InApp {
		return "In App"
	}
	return ""
}

func (id *IssueDetails) HasStackDetails() bool { return len(id.StackDetails) > 0 }

func (id *IssueDetails) StackVisible() []StackDetail {
	if len(id.StackDetails) > 5 {
		return id.StackDetails[:5]
	}
	return id.StackDetails
}

func (id *IssueDetails) StackHidden() []StackDetail {
	if len(id.StackDetails) > 5 {
		return id.StackDetails[5:]
	}
	return nil
}

func GetStackDetails(event *IssueEvent) []StackDetail {
	if len(event.ExceptionFramesAbsPath) == 0 {
		return []StackDetail{}
	}

	res := make([]StackDetail, 0, len(event.ExceptionFramesAbsPath))
	for i := range event.ExceptionFramesAbsPath {
		res = append(res, StackDetail{
			Filepath: event.ExceptionFramesAbsPath[i],
		})
	}

	for i := range res {
		res[i].FunctionName = event.ExceptionFramesFunction[i]
		res[i].LineNo = event.ExceptionFramesLineno[i]
		if event.ExceptionFramesInApp[i] == 1 {
			res[i].InApp = true
		}
	}

	slices.Reverse(res)

	return res
}

func (id *IssueDetails) ListTagValues(tag string) []FieldValueNum {
	var res []FieldValueNum
	for _, tv := range id.TagValueNum {
		if tv.Tag == tag {
			res = append(res, tv)
		}
	}
	return res
}

func ListTagValues(tag string, tv []FieldValueNum) []FieldValueNum {
	var res []FieldValueNum
	for _, t := range tv {
		if t.Tag == tag {
			res = append(res, t)
		}
	}
	return res
}

func (t *FieldValueNum) PercentsFormatted() string {
	return strconv.Itoa(int(math.Floor(t.PercentsOfTotal)))
}

type ProjectDetailsRequest struct {
	Issues    IssuesType
	Period    string
	Start     string
	End       string
	ProjectID int
	Page      int
}

type IssuesType string

const (
	IssuesTypeAll IssuesType = "all"
	IssuesTypeNew IssuesType = "new"
)

var AllowedIssuesTypes = [...]IssuesType{IssuesTypeAll, IssuesTypeNew}

type ProjectDetails struct {
	Project     *Project     `json:"project"`
	Assignments *Assignments `json:"assignments"`
	Period      string       `json:"period"`
	Issues      IssuesType   `json:"issues"`
	Teammates   []Teammate   `json:"teammates"`
	Page        int          `json:"page"`
}

func (pd *ProjectDetails) AllLength() string {
	if pd.Project.AllLength == 0 {
		return ""
	}
	return NumFormatted(pd.Project.AllLength)
}

func (pd *ProjectDetails) NewLength() string {
	if pd.Project.NewLength == 0 {
		return ""
	}
	return NumFormatted(pd.Project.NewLength)
}

// ListProjectsCriteria is a criteria to list projects.
type ListProjectsCriteria struct {
	Name   string `json:"name"    schema:"name"`
	TeamID int    `json:"team_id" schema:"team"`
}

// IsEmpty returns true if the criteria is empty.
func (c *ListProjectsCriteria) IsEmpty() bool {
	return c.TeamID == 0 && c.Name == ""
}

// CreateProjectRequest is a request to create a new project.
// platform=go&highPriority=false&threshold=10&condition=1&timeframe=1&projectName=golang&team=1.
type CreateProjectRequest struct {
	Platform     string `schema:"platform,required"    validate:"required,gt=0,lt=32"`
	ProjectName  string `schema:"projectName,required" validate:"required,gt=0,lt=32"`
	Threshold    int    `schema:"threshold,required"   validate:"required,gt=0,lt=1000000"`
	Condition    int    `schema:"condition,required"   validate:"required,gt=0,lt=3"`
	Timeframe    int    `schema:"timeframe,required"   validate:"required,gt=0,lt=8"`
	TeamID       int    `schema:"team,required"        validate:"required,gt=0"`
	HighPriority bool   `schema:"highPriority"`
}

// ProjectInfo is a representation of a project.
type ProjectInfo struct {
	Name     string `json:"name"`
	DSN      string `json:"dsn"`
	Platform string `json:"platform"`
	ID       int    `json:"id"`
}

// Team is a representation of a team in the system.
type Team struct {
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	ID        int       `json:"id"`
	OwnerID   int       `json:"owner_id"`
}

// TeamStore encapsulates the team storage.
type TeamStore interface {
	// CreateTeam creates a new team.
	CreateTeam(ctx context.Context, team Team) error
	// ListTeams returns a list of teams for the given user.
	ListTeams(ctx context.Context, userID int) ([]Team, error)
	// ListTeammates returns a list of teammates for the given team.
	ListTeammates(ctx context.Context, teamIDs []int) ([]Teammate, error)
	// AddUserToTeam adds a user to a team.
	AddUserToTeam(ctx context.Context, createdAt time.Time, userID int64, teamID int) error
}

// Platform represents the platform of the project.
type Platform int8

// String returns the string representation of the platform.
func (p Platform) String() string {
	switch p {
	case PlatformGolang:
		return "Go"
	case PlatformRust:
		return "Rust"
	default:
		return "unknown"
	}
}

// PlatformByName returns the platform by name.
func PlatformByName(name string) Platform {
	switch name {
	case "go":
		return PlatformGolang
	case "rust":
		return PlatformRust
	default:
		return 0
	}
}

func GetSDKID(name string) uint8 {
	switch name {
	case "sentry.go":
		return 1
	case "sentry.rust":
		return 2
	default:
		return 0
	}
}

type EventType = uint8

const (
	EventTypeException EventType = iota + 1
)

// DashboardData returns the data for frontend dashboard (24h period).
func (e EventsList) DashboardData(now func() time.Time) string {
	return e.DashboardDataForPeriod(now, "24h")
}

// DashboardDataForPeriod returns the data for frontend dashboard adapted to the given period.
// Returns JSON array of [timestamps, counts] like: [[t1,t2,t3...], [c1,c2,c3...]].
func (e EventsList) DashboardDataForPeriod(now func() time.Time, period string) string {
	if period == "" {
		period = "24h"
	}

	duration, err := ParseDuration(period)
	if err != nil {
		duration = 24 * time.Hour
	}

	// Determine interval based on period
	var interval time.Duration
	timeNow := now().UTC()

	switch {
	case duration <= 24*time.Hour:
		interval = time.Hour
	case duration <= 7*24*time.Hour:
		interval = 6 * time.Hour
	case duration <= 30*24*time.Hour:
		interval = 24 * time.Hour
	default:
		interval = 24 * time.Hour
	}

	endTime := timeNow.Truncate(time.Hour).Add(time.Hour) // Round up to next hour boundary
	startTime := endTime.Add(-duration)

	if duration < 6*time.Hour {
		startTime = endTime.Add(-24 * time.Hour)
	}

	// Align start/end to interval boundaries
	startTime = startTime.Truncate(interval)
	endTime = endTime.Truncate(interval)
	if endTime.Before(timeNow) {
		endTime = endTime.Add(interval)
	}

	// Calculate number of buckets
	numPoints := int(endTime.Sub(startTime) / interval)
	if numPoints > 100 {
		numPoints = 100
		interval = endTime.Sub(startTime) / time.Duration(numPoints)
		startTime = startTime.Truncate(interval)
	}

	// Create map of events by timestamp for quick lookup (ClickHouse returns hourly data)
	hourlyEventMap := make(map[int64]int, len(e))
	for _, event := range e {
		hourKey := event.TS.Truncate(time.Hour).Unix()
		hourlyEventMap[hourKey] += event.Count
	}

	// Build arrays of timestamps and counts, aggregating hourly data into intervals
	timestamps := make([]int64, numPoints)
	counts := make([]int, numPoints)

	for i := range numPoints {
		bucketStart := startTime.Add(time.Duration(i) * interval)
		bucketEnd := bucketStart.Add(interval)
		timestamps[i] = bucketStart.Unix()

		// Aggregate all hourly events that fall in this bucket
		for hourTS, hourCount := range hourlyEventMap {
			hourTime := time.Unix(hourTS, 0).UTC()
			if (hourTime.Equal(bucketStart) || hourTime.After(bucketStart)) && hourTime.Before(bucketEnd) {
				counts[i] += hourCount
			}
		}
	}

	// Format as JSON: [[timestamps...], [counts...]]
	var result strings.Builder
	result.WriteString("[[")
	for i, ts := range timestamps {
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(strconv.FormatInt(ts, 10))
	}
	result.WriteString("],[")
	for i, count := range counts {
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(strconv.Itoa(count))
	}
	result.WriteString("]]")

	return result.String()
}

// TotalErrors returns the total number of errors.
func (e EventsList) TotalErrors() string {
	total := 0
	for i := range e {
		total += e[i].Count
	}
	switch {
	case total > 1000000:
		return fmt.Sprintf("%.1fm", float64(total)/1000000)
	case total > 1000:
		if total%1000 == 0 {
			return fmt.Sprintf("%dk", total/1000)
		}
		return fmt.Sprintf("%.1fk", float64(total)/1000)
	default:
		return strconv.Itoa(total)
	}
}

func NumFormatted[T int | int64 | uint64 | float64](num T) string {
	switch {
	case num > 1000000:
		return fmt.Sprintf("%.1fm", float64(num)/1000000)
	case num > 1000:
		return fmt.Sprintf("%.1fk", float64(num)/1000)
	default:
		return fmt.Sprintf("%v", num)
	}
}

var unitMap = map[string]time.Duration{
	"h": time.Hour,
	"m": time.Minute,
	"s": time.Second,
	"d": time.Hour * 24,
	"w": time.Hour * 24 * 7,
}

// ParseDuration parses a string like "2h", "2d", "1w" into a time.Duration.
// The returned duration is always positive.
func ParseDuration(input string) (time.Duration, error) {
	if input == "" {
		return 0, nil
	}

	unit := input[len(input)-1:]
	duration, err := strconv.Atoi(input[:len(input)-1])
	if err != nil {
		return 0, fmt.Errorf("warnly: parse duration atoi: %w", err)
	}

	d, ok := unitMap[unit]
	if !ok {
		return 0, fmt.Errorf("unknown unit %s", unit)
	}

	if duration < 0 {
		duration = -duration
	}

	return time.Duration(duration) * d, nil
}

// ParseTimeRange parses a time range from two strings in the format "2006-01-02T15:04:05".
func ParseTimeRange(start, end string) (time.Time, time.Time, error) {
	const layout = "2006-01-02T15:04:05"

	startTime, err := time.Parse(layout, start)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("warnly: parse time range start: %w", err)
	}

	endTime, err := time.Parse(layout, end)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("warnly: parse time range end: %w", err)
	}

	return startTime, endTime, nil
}
