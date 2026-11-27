package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// ArchivedCopilotClient reads only selected public toggle values. It never
// infers a missing value and never performs an update.
type ArchivedCopilotClient struct{ rest *api.RESTClient }

func NewArchivedCopilotClient(host string) (*ArchivedCopilotClient, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, fmt.Errorf("configure archived Copilot reader")
	}
	return &ArchivedCopilotClient{rest: rest}, nil
}

func (c *ArchivedCopilotClient) Read(repo repository.Repository, selected settings.ArchivedCopilotToggles) (settings.ArchivedCopilotToggles, error) {
	var result settings.ArchivedCopilotToggles
	response, err := c.rest.Request(http.MethodGet, copilotInstructionsPath(repo), nil)
	if err != nil {
		return result, fmt.Errorf("read archived Copilot settings failed")
	}
	defer response.Body.Close()
	var fields map[string]json.RawMessage
	if response.StatusCode != http.StatusOK || decodeCopilotSettings(response.Body, &fields) != nil {
		return result, fmt.Errorf("archived Copilot response unavailable")
	}
	for _, field := range []struct {
		key         string
		selected    *bool
		destination **bool
	}{
		{"custom_instructions_enabled", selected.Instructions, &result.Instructions},
		{"mcp_enabled", selected.MCP, &result.MCP},
		{"cross_repo_context_enabled", selected.PublicSearch, &result.PublicSearch},
	} {
		if field.selected != nil && (json.Unmarshal(fields[field.key], field.destination) != nil || *field.destination == nil) {
			return result, fmt.Errorf("missing or invalid archived Copilot %s", field.key)
		}
	}
	return result, nil
}
