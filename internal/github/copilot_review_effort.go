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

// CopilotReviewEffortClient manages only review effort, never review activation.
type CopilotReviewEffortClient struct{ rest *api.RESTClient }

func NewCopilotReviewEffortClient(host string) (*CopilotReviewEffortClient, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &copilotReviewEffortPreWriteFailure{fmt.Errorf("configure Copilot review effort API")}
	}
	return &CopilotReviewEffortClient{rest: rest}, nil
}

type copilotReviewEffortState struct {
	effort            settings.CopilotReviewEffortObservation
	severity          string
	instructions, mcp bool
}

func (c *CopilotReviewEffortClient) read(repo repository.Repository) (copilotReviewEffortState, error) {
	response, err := c.rest.Request(http.MethodGet, copilotInstructionsPath(repo), nil)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) {
			return copilotReviewEffortState{}, fmt.Errorf("read Copilot review effort: HTTP %d", httpErr.StatusCode)
		}
		return copilotReviewEffortState{}, fmt.Errorf("read Copilot review effort failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return copilotReviewEffortState{}, fmt.Errorf("unexpected Copilot review effort read status %d", response.StatusCode)
	}
	var raw struct {
		Effort *struct {
			Configured json.RawMessage `json:"configured"`
			Effective  *string         `json:"effective"`
		} `json:"review_effort"`
		Available    *[]json.RawMessage `json:"available_review_efforts"`
		Severity     *string            `json:"severity_filter"`
		Instructions *bool              `json:"custom_instructions_enabled"`
		MCP          *bool              `json:"mcp_enabled"`
	}
	decoder := json.NewDecoder(response.Body)
	if decoder.Decode(&raw) != nil || raw.Effort == nil || len(raw.Effort.Configured) == 0 || raw.Effort.Effective == nil || raw.Available == nil || raw.Severity == nil || raw.Instructions == nil || raw.MCP == nil {
		return copilotReviewEffortState{}, fmt.Errorf("incomplete or invalid Copilot review effort settings")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return copilotReviewEffortState{}, fmt.Errorf("invalid trailing Copilot review effort response")
	}
	var configured *string
	if string(raw.Effort.Configured) != "null" {
		var value string
		if json.Unmarshal(raw.Effort.Configured, &value) != nil {
			return copilotReviewEffortState{}, fmt.Errorf("invalid configured Copilot review effort")
		}
		configured = &value
	}
	available := make([]string, 0, len(*raw.Available))
	for _, item := range *raw.Available {
		var value *string
		if json.Unmarshal(item, &value) != nil || value == nil {
			return copilotReviewEffortState{}, fmt.Errorf("invalid available Copilot review effort")
		}
		available = append(available, *value)
	}
	switch *raw.Severity {
	case "no_filter", "low", "low_and_medium":
	default:
		return copilotReviewEffortState{}, fmt.Errorf("unsupported Copilot severity filter")
	}
	effort, err := settings.NewCopilotReviewEffortObservation(configured, *raw.Effort.Effective, available)
	if err != nil {
		return copilotReviewEffortState{}, err
	}
	return copilotReviewEffortState{effort: effort, severity: *raw.Severity, instructions: *raw.Instructions, mcp: *raw.MCP}, nil
}

func (c *CopilotReviewEffortClient) Read(repo repository.Repository) (settings.CopilotReviewEffortObservation, error) {
	state, err := c.read(repo)
	return state.effort, err
}

type copilotReviewEffortPreWriteFailure struct{ error }

func (*copilotReviewEffortPreWriteFailure) ReadFailedBeforeWrite() {}

// Update validates fresh availability and independently reads back observable
// configuration. A null configured value cannot prove erasure of a hidden override.
func (c *CopilotReviewEffortClient) Update(repo repository.Repository, desired string) (bool, error) {
	if err := settings.ValidateCopilotReviewEffort(desired); err != nil {
		return false, &copilotReviewEffortPreWriteFailure{err}
	}
	before, err := c.read(repo)
	if err != nil {
		return false, &copilotReviewEffortPreWriteFailure{err}
	}
	if err := before.effort.ValidateDesired(desired); err != nil {
		return false, &copilotReviewEffortPreWriteFailure{err}
	}
	current, err := before.effort.Value()
	if err != nil {
		return false, &copilotReviewEffortPreWriteFailure{err}
	}
	if current == desired {
		return false, nil
	}
	value := desired
	if desired == "default" {
		value = "inherit"
	}
	body, _ := json.Marshal(map[string]string{"review_effort": value})
	response, err := c.rest.Request(http.MethodPatch, copilotInstructionsPath(repo), bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("write Copilot review effort failed (write effects may already have applied)")
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected Copilot review effort write status %d (write effects may already have applied)", response.StatusCode)
	}
	after, err := c.read(repo)
	if err != nil {
		return false, fmt.Errorf("verify Copilot review effort failed (write effects may already have applied)")
	}
	actual, err := after.effort.Value()
	if err != nil || actual != desired {
		return false, fmt.Errorf("Copilot review effort was not retained (write effects may already have applied)")
	}
	if before.severity != after.severity || before.instructions != after.instructions || before.mcp != after.mcp {
		return false, fmt.Errorf("unmanaged Copilot settings changed during review effort update (write effects may already have applied)")
	}
	return true, nil
}
