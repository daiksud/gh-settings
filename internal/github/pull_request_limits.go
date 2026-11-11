package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type PullRequestLimitsClient struct {
	rest  *api.RESTClient
	pause func(time.Duration)
}

func NewPullRequestLimitsClient(host string) (*PullRequestLimitsClient, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &pullRequestLimitsPreWriteFailure{fmt.Errorf("configure pull request limits API")}
	}
	return &PullRequestLimitsClient{rest: rest, pause: time.Sleep}, nil
}
func pullRequestLimitsPath(repo repository.Repository) string {
	return fmt.Sprintf("repos/%s/%s/interaction-limits/pulls/creation-cap", repo.Owner, repo.Name)
}
func (c *PullRequestLimitsClient) Read(repo repository.Repository) (settings.PullRequestLimitsObservation, error) {
	response, err := c.rest.Request(http.MethodGet, pullRequestLimitsPath(repo), nil)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) {
			return settings.PullRequestLimitsObservation{}, fmt.Errorf("read pull request limits: HTTP %d", httpErr.StatusCode)
		}
		return settings.PullRequestLimitsObservation{}, fmt.Errorf("read pull request limits failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return settings.PullRequestLimitsObservation{}, fmt.Errorf("unexpected pull request limits read status %d", response.StatusCode)
	}
	var raw struct {
		Enabled *bool `json:"enabled"`
		Maximum *int  `json:"max_open_pull_requests"`
		Drafts  *bool `json:"include_drafts"`
	}
	dec := json.NewDecoder(response.Body)
	if dec.Decode(&raw) != nil || raw.Enabled == nil || raw.Maximum == nil || raw.Drafts == nil {
		return settings.PullRequestLimitsObservation{}, fmt.Errorf("incomplete or invalid pull request limits")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return settings.PullRequestLimitsObservation{}, fmt.Errorf("invalid trailing pull request limits response")
	}
	return settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: *raw.Enabled, Maximum: *raw.Maximum, IncludeDrafts: *raw.Drafts})
}

type pullRequestLimitsPreWriteFailure struct{ error }

func (*pullRequestLimitsPreWriteFailure) ReadFailedBeforeWrite() {}
func (c *PullRequestLimitsClient) Update(repo repository.Repository, d settings.PullRequestLimitsDeclaration) (bool, error) {
	if err := d.Validate(); err != nil {
		return false, &pullRequestLimitsPreWriteFailure{err}
	}
	if d.Empty() {
		return false, nil
	}
	observed, err := c.Read(repo)
	if err != nil {
		return false, &pullRequestLimitsPreWriteFailure{err}
	}
	before, _ := observed.Values()
	expected := before
	if d.Enabled != nil {
		expected.Enabled = *d.Enabled
	}
	if d.Maximum != nil {
		expected.Maximum = *d.Maximum
	}
	if d.IncludeDrafts != nil {
		expected.IncludeDrafts = *d.IncludeDrafts
	}
	if before == expected {
		return false, nil
	}
	// enabled is required by PATCH even when the declaration owns only another control.
	payload := map[string]any{"enabled": expected.Enabled}
	if d.Maximum != nil {
		payload["max_open_pull_requests"] = *d.Maximum
	}
	if d.IncludeDrafts != nil {
		payload["include_drafts"] = *d.IncludeDrafts
	}
	body, _ := json.Marshal(payload)
	response, err := c.rest.Request(http.MethodPatch, pullRequestLimitsPath(repo), bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("write pull request limits failed (write effects may already have applied)")
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected pull request limits write status %d (write effects may already have applied)", response.StatusCode)
	}
	// Only well-formed observations equal to the complete pre-write baseline may
	// be stale. Never retry writes, denials, malformed reads or companion drift.
	for attempt := 0; attempt < 3; attempt++ {
		observed, err = c.Read(repo)
		if err != nil {
			return false, fmt.Errorf("verify pull request limits failed (write effects may already have applied)")
		}
		after, _ := observed.Values()
		if after == expected {
			return true, nil
		}
		if after != before {
			return false, fmt.Errorf("pull request limits readback differs from requested or preserved state (write effects may already have applied)")
		}
		if attempt < 2 {
			pause := c.pause
			if pause == nil {
				pause = time.Sleep
			}
			pause(time.Second)
		}
	}
	return false, fmt.Errorf("pull request limits were not confirmed after three reads (write effects may already have applied)")
}
