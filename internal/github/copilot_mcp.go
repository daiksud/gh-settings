package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// CopilotContextClient manages the MCP and public repository search settings.
type CopilotContextClient struct{ rest *api.RESTClient }

func NewCopilotContextClient(host string) (*CopilotContextClient, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &copilotContextPreWriteFailure{fmt.Errorf("configure Copilot context API")}
	}
	return &CopilotContextClient{rest: rest}, nil
}

// Preserve the complete known endpoint state without applying the managed
// review-effort control's narrower UI choices to these unowned API values.
type copilotContextState struct {
	context             settings.CopilotContextObservation
	instructions        bool
	configured          *string
	effective, severity string
	available           []string
}

func (c *CopilotContextClient) read(repo repository.Repository) (copilotContextState, error) {
	response, err := c.rest.Request(http.MethodGet, copilotInstructionsPath(repo), nil)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) {
			return copilotContextState{}, fmt.Errorf("read Copilot context: HTTP %d", httpErr.StatusCode)
		}
		return copilotContextState{}, fmt.Errorf("read Copilot context failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return copilotContextState{}, fmt.Errorf("unexpected Copilot context read status %d", response.StatusCode)
	}
	var raw struct {
		PublicSearch optionalSearchSetting `json:"cross_repo_context_enabled"`
		MCP          *bool                 `json:"mcp_enabled"`
		Instructions *bool                 `json:"custom_instructions_enabled"`
		Severity     *string               `json:"severity_filter"`
		Effort       *struct {
			Configured json.RawMessage `json:"configured"`
			Effective  *string         `json:"effective"`
		} `json:"review_effort"`
		Available *[]json.RawMessage `json:"available_review_efforts"`
	}
	if decodeCopilotSettings(response.Body, &raw) != nil || raw.MCP == nil || raw.Instructions == nil || raw.Severity == nil || raw.Effort == nil || len(raw.Effort.Configured) == 0 || raw.Effort.Effective == nil || raw.Available == nil {
		return copilotContextState{}, fmt.Errorf("incomplete or invalid Copilot context settings")
	}
	state := copilotContextState{context: settings.CopilotContextObservation{MCP: raw.MCP, PublicSearch: raw.PublicSearch.Value}, instructions: *raw.Instructions, severity: *raw.Severity, effective: *raw.Effort.Effective, available: make([]string, 0, len(*raw.Available))}
	if string(raw.Effort.Configured) != "null" {
		var value string
		if json.Unmarshal(raw.Effort.Configured, &value) != nil || strings.TrimSpace(value) == "" {
			return copilotContextState{}, fmt.Errorf("invalid configured Copilot review effort")
		}
		state.configured = &value
	}
	if strings.TrimSpace(state.effective) == "" {
		return copilotContextState{}, fmt.Errorf("invalid effective Copilot review effort")
	}
	for _, item := range *raw.Available {
		var value *string
		if json.Unmarshal(item, &value) != nil || value == nil || strings.TrimSpace(*value) == "" || slices.Contains(state.available, *value) {
			return copilotContextState{}, fmt.Errorf("invalid available Copilot review effort")
		}
		state.available = append(state.available, *value)
	}
	switch state.severity {
	case "no_filter", "low", "low_and_medium":
	default:
		return copilotContextState{}, fmt.Errorf("unsupported Copilot severity filter")
	}
	return state, nil
}

func (c *CopilotContextClient) Read(repo repository.Repository) (settings.CopilotContextObservation, error) {
	state, err := c.read(repo)
	if err != nil {
		return settings.CopilotContextObservation{}, err
	}
	return state.context, nil
}

type copilotContextPreWriteFailure struct{ error }

func (*copilotContextPreWriteFailure) ReadFailedBeforeWrite() {}

// The response is shared by independently managed settings. Decode exact field names
// so a duplicate or case-insensitive alias cannot be used to verify a write.
func decodeCopilotSettings(body io.Reader, out any) error {
	const limit = 1 << 20
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	var fields map[string]json.RawMessage
	if err != nil || len(data) > limit || !validEnvironmentJSONUnicode(data) || webhookJSON(data, &fields) != nil || fields == nil ||
		hasActionsFieldAlias(fields, "custom_instructions_enabled", "mcp_enabled", "cross_repo_context_enabled", "review_effort", "available_review_efforts", "severity_filter") {
		return fmt.Errorf("invalid or ambiguous Copilot settings")
	}
	if raw := fields["review_effort"]; raw != nil {
		var effort map[string]json.RawMessage
		if json.Unmarshal(raw, &effort) == nil && hasActionsFieldAlias(effort, "configured", "effective") {
			return fmt.Errorf("ambiguous Copilot review effort")
		}
	}
	return json.Unmarshal(data, out)
}

// An omitted field provides no evidence of the public search setting. A present
// field must contain a native boolean.
type optionalSearchSetting struct{ Value *bool }

func (s *optionalSearchSetting) UnmarshalJSON(raw []byte) error {
	switch string(raw) {
	case "false", "true":
		value := string(raw) == "true"
		s.Value = &value
		return nil
	default:
		return fmt.Errorf("invalid Copilot public search boolean")
	}
}

func (c *CopilotContextClient) UpdateMCP(repo repository.Repository, desired bool) (bool, error) {
	return c.update(repo, settings.CopilotMCPPath, "mcp_enabled", desired)
}
func (c *CopilotContextClient) UpdatePublicSearch(repo repository.Repository, desired bool) (bool, error) {
	return c.update(repo, settings.CopilotPublicSearchPath, "cross_repo_context_enabled", desired)
}

func (c *CopilotContextClient) update(repo repository.Repository, field, apiField string, desired bool) (bool, error) {
	before, err := c.read(repo)
	if err != nil {
		return false, &copilotContextPreWriteFailure{err}
	}
	current, err := before.context.Value(field)
	if err != nil {
		return false, &copilotContextPreWriteFailure{err}
	}
	if current == desired {
		return false, nil
	}
	body, _ := json.Marshal(map[string]bool{apiField: desired})
	response, err := c.rest.Request(http.MethodPatch, copilotInstructionsPath(repo), bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("write %s failed (write effects may already have applied)", field)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected %s write status %d (write effects may already have applied)", field, response.StatusCode)
	}
	after, err := c.read(repo)
	if err != nil {
		return false, fmt.Errorf("verify %s failed (write effects may already have applied)", field)
	}
	expected := before
	if field == settings.CopilotMCPPath {
		expected.context.MCP = &desired
	} else {
		expected.context.PublicSearch = &desired
	}
	if !reflect.DeepEqual(after, expected) {
		return false, fmt.Errorf("%s or companion settings were not retained (write effects may already have applied)", field)
	}
	return true, nil
}
