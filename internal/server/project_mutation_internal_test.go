package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vk-rv/warnly/internal/warnly"
)

type mutationProjectService struct {
	warnly.ProjectService

	err   error
	calls int
}

func (s *mutationProjectService) AssignIssue(context.Context, *warnly.AssignIssueRequest) error {
	s.calls++
	return s.err
}

func (s *mutationProjectService) DeleteAssignment(context.Context, *warnly.UnassignIssueRequest) error {
	s.calls++
	return s.err
}

func (s *mutationProjectService) CreateMessage(context.Context, *warnly.CreateMessageRequest) error {
	s.calls++
	return s.err
}

func (s *mutationProjectService) DeleteMessage(context.Context, *warnly.DeleteMessageRequest) error {
	s.calls++
	return s.err
}

func TestProjectMutationsDoNotReloadData(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"assign", "unassign", "post", "delete"} {
		for _, fail := range []bool{false, true} {
			t.Run(action+map[bool]string{false: "/success", true: "/failure"}[fail], func(t *testing.T) {
				t.Parallel()

				svc := &mutationProjectService{}
				if fail {
					svc.err = errors.New("write failed")
				}
				h := NewProjectHandler(svc, slog.Default())
				handler := h.AssignIssue
				method := http.MethodPost
				switch action {
				case "unassign":
					handler = h.DeleteAssignment
					method = http.MethodDelete
				case "post":
					handler = h.PostMessage
				case "delete":
					handler = h.DeleteMessage
					method = http.MethodDelete
				}
				ctx := context.WithValue(t.Context(), userContextKey, warnly.User{ID: 7})
				r := httptest.NewRequestWithContext(ctx, method, "/?issues=all&period=14d&page=2", strings.NewReader("user_id=7&content=hello"))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				r.SetPathValue("project_id", "1")
				r.SetPathValue("issue_id", "2")
				r.SetPathValue("message_id", "3")
				w := httptest.NewRecorder()
				// Any read method on the embedded nil service panics.
				handler(w, r)
				want := http.StatusNoContent
				if fail {
					want = http.StatusInternalServerError
				}
				if w.Code != want {
					t.Fatalf("status = %d, want %d", w.Code, want)
				}
				if !fail && w.Body.Len() != 0 {
					t.Fatalf("unexpected response: %s", w.Body.String())
				}
				if svc.calls != 1 {
					t.Fatalf("mutation calls = %d, want 1", svc.calls)
				}
			})
		}
	}
}
