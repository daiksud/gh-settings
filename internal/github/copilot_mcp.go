package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// CopilotMCPClient manages only the stored MCP tools toggle for Copilot code review. It does
// not activate reviews or manage MCP server configuration files.
type CopilotMCPClient struct{ rest *api.RESTClient }

func NewCopilotMCPClient(host string) (*CopilotMCPClient, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &copilotMCPPreWriteFailure{fmt.Errorf("configure Copilot MCP API")}
	}
	return &CopilotMCPClient{rest: rest}, nil
}

// Preserve the complete known endpoint state without applying the managed
// review-effort control's narrower UI choices to these unowned API values.
type copilotMCPState struct {
	enabled, instructions bool
	configured            *string
	effective, severity   string
	available             []string
}

func (c *CopilotMCPClient) read(repo repository.Repository) (copilotMCPState, error) {
	response, err := c.rest.Request(http.MethodGet, copilotInstructionsPath(repo), nil)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) {
			return copilotMCPState{}, fmt.Errorf("read Copilot MCP: HTTP %d", httpErr.StatusCode)
		}
		return copilotMCPState{}, fmt.Errorf("read Copilot MCP failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return copilotMCPState{}, fmt.Errorf("unexpected Copilot MCP read status %d", response.StatusCode)
	}
	var raw struct {
		MCP          *bool   `json:"mcp_enabled"`
		Instructions *bool   `json:"custom_instructions_enabled"`
		Severity     *string `json:"severity_filter"`
		Effort       *struct {
			Configured json.RawMessage `json:"configured"`
			Effective  *string         `json:"effective"`
		} `json:"review_effort"`
		Available *[]json.RawMessage `json:"available_review_efforts"`
	}
	decoder := json.NewDecoder(response.Body)
	if decoder.Decode(&raw) != nil || raw.MCP == nil || raw.Instructions == nil || raw.Severity == nil || raw.Effort == nil || len(raw.Effort.Configured) == 0 || raw.Effort.Effective == nil || raw.Available == nil {
		return copilotMCPState{}, fmt.Errorf("incomplete or invalid Copilot MCP settings")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return copilotMCPState{}, fmt.Errorf("invalid trailing Copilot MCP response")
	}
	state := copilotMCPState{enabled: *raw.MCP, instructions: *raw.Instructions, severity: *raw.Severity, effective: *raw.Effort.Effective, available: make([]string, 0, len(*raw.Available))}
	if string(raw.Effort.Configured) != "null" {
		var value string
		if json.Unmarshal(raw.Effort.Configured, &value) != nil || strings.TrimSpace(value) == "" {
			return copilotMCPState{}, fmt.Errorf("invalid configured Copilot review effort")
		}
		state.configured = &value
	}
	if strings.TrimSpace(state.effective) == "" {
		return copilotMCPState{}, fmt.Errorf("invalid effective Copilot review effort")
	}
	for _, item := range *raw.Available {
		var value *string
		if json.Unmarshal(item, &value) != nil || value == nil || strings.TrimSpace(*value) == "" || slices.Contains(state.available, *value) {
			return copilotMCPState{}, fmt.Errorf("invalid available Copilot review effort")
		}
		state.available = append(state.available, *value)
	}
	switch state.severity {
	case "no_filter", "low", "low_and_medium":
	default:
		return copilotMCPState{}, fmt.Errorf("unsupported Copilot severity filter")
	}
	return state, nil
}

func (c *CopilotMCPClient) Read(repo repository.Repository) (settings.CopilotMCPObservation, error) {
	state, err := c.read(repo)
	if err != nil {
		return settings.CopilotMCPObservation{}, err
	}
	return settings.NewCopilotMCPObservation(state.enabled), nil
}

type copilotMCPPreWriteFailure struct{ error }

func (*copilotMCPPreWriteFailure) ReadFailedBeforeWrite() {}

// Update reads fresh state and verifies both the requested value and known
// unowned fields with an independent GET. Errors after PATCH are uncertain.
func (c *CopilotMCPClient) Update(repo repository.Repository, desired bool) (bool, error) {
	before, err := c.read(repo)
	if err != nil {
		return false, &copilotMCPPreWriteFailure{err}
	}
	if before.enabled == desired {
		return false, nil
	}
	body, _ := json.Marshal(map[string]bool{"mcp_enabled": desired})
	response, err := c.rest.Request(http.MethodPatch, copilotInstructionsPath(repo), bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("write Copilot MCP failed (write effects may already have applied)")
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected Copilot MCP write status %d (write effects may already have applied)", response.StatusCode)
	}
	after, err := c.read(repo)
	if err != nil {
		return false, fmt.Errorf("verify Copilot MCP failed (write effects may already have applied)")
	}
	if after.enabled != desired {
		return false, fmt.Errorf("Copilot MCP was not retained (write effects may already have applied)")
	}
	sameConfigured := before.configured == nil && after.configured == nil || before.configured != nil && after.configured != nil && *before.configured == *after.configured
	if !sameConfigured || before.effective != after.effective || !slices.Equal(before.available, after.available) || before.instructions != after.instructions || before.severity != after.severity {
		return false, fmt.Errorf("unmanaged Copilot settings changed during MCP update (write effects may already have applied)")
	}
	return true, nil
}
