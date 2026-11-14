package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func (c *ArchivedCopilotClient) Read(repo repository.Repository, instructions, mcp bool) (settings.ArchivedCopilotToggles, error) {
	var result settings.ArchivedCopilotToggles
	response, err := c.rest.Request(http.MethodGet, copilotInstructionsPath(repo), nil)
	if err != nil {
		return result, fmt.Errorf("read archived Copilot settings failed")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 || response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("archived Copilot response unavailable")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return result, fmt.Errorf("invalid archived Copilot response")
	}
	seen := map[string]bool{}
	for decoder.More() {
		key, err := decoder.Token()
		name, ok := key.(string)
		if err != nil || !ok || seen[name] {
			return result, fmt.Errorf("invalid archived Copilot response")
		}
		seen[name] = true
		var raw json.RawMessage
		if decoder.Decode(&raw) != nil {
			return result, fmt.Errorf("invalid archived Copilot response")
		}
		// encoding/json matches struct keys without regard to case. Reject
		// aliases so a conflicting field cannot masquerade as evidence.
		if (strings.EqualFold(name, "custom_instructions_enabled") && name != "custom_instructions_enabled") || (strings.EqualFold(name, "mcp_enabled") && name != "mcp_enabled") {
			return result, fmt.Errorf("invalid archived Copilot response")
		}
		if name == "custom_instructions_enabled" && instructions {
			if json.Unmarshal(raw, &result.Instructions) != nil || result.Instructions == nil {
				return result, fmt.Errorf("invalid archived Copilot instructions")
			}
		}
		if name == "mcp_enabled" && mcp {
			if json.Unmarshal(raw, &result.MCP) != nil || result.MCP == nil {
				return result, fmt.Errorf("invalid archived Copilot MCP")
			}
		}
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return result, fmt.Errorf("invalid archived Copilot response")
	}
	if _, err := decoder.Token(); err != io.EOF || instructions && result.Instructions == nil || mcp && result.MCP == nil {
		return result, fmt.Errorf("incomplete archived Copilot response")
	}
	return result, nil
}
