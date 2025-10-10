package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/vk-rv/warnly/internal/warnly"
)

const (
	// htmxHeader is the HTTP header used by HTMX to indicate an HTMX request.
	htmxHeader = "Hx-Request"
	// htmxTarget is the HTTP header used by HTMX to indicate the target element for the response.
	htmxTarget = "Hx-Target"
)

func projectDecodeValid(r *http.Request) (*warnly.CreateProjectRequest, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("project decode valid: parse form: %w", err)
	}

	req := &warnly.CreateProjectRequest{}

	req.ProjectName = r.FormValue("projectName")
	req.Platform = r.FormValue("platform")
	teamID, err := strconv.Atoi(r.FormValue("team"))
	if err != nil {
		return nil, fmt.Errorf("project decode valid: parse team_id: %w", err)
	}
	req.TeamID = teamID

	if req.ProjectName == "" || req.Platform == "" {
		return nil, errors.New("project decode valid: missing required fields")
	}

	return req, nil
}
