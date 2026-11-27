package github

import (
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedCopilotRead(t *testing.T) {
	for _, tc := range []struct {
		name, body              string
		status                  int
		instructions, mcp, fail bool
	}{
		{"both", `{"custom_instructions_enabled":true,"mcp_enabled":false}`, 200, true, true, false},
		{"instructions only", `{"custom_instructions_enabled":false}`, 200, true, false, false},
		{"mcp only", `{"mcp_enabled":true}`, 200, false, true, false},
		{"unselected companion invalid", `{"custom_instructions_enabled":true,"mcp_enabled":null}`, 200, true, false, false},
		{"missing instructions", `{"mcp_enabled":false}`, 200, true, false, true},
		{"missing mcp", `{"custom_instructions_enabled":true}`, 200, false, true, true},
		{"null", `{"custom_instructions_enabled":null}`, 200, true, false, true},
		{"wrong type", `{"mcp_enabled":"false"}`, 200, false, true, true},
		{"duplicate", `{"custom_instructions_enabled":false,"custom_instructions_enabled":true}`, 200, true, false, true},
		{"case alias", `{"custom_instructions_enabled":true,"CUSTOM_INSTRUCTIONS_ENABLED":false}`, 200, true, false, true},
		{"mcp case alias", `{"MCP_ENABLED":false,"mcp_enabled":true}`, 200, false, true, true},
		{"trailing", `{"mcp_enabled":false} {}`, 200, false, true, true},
		{"array", `[]`, 200, true, true, true},
		{"malformed", `{"mcp_enabled":false`, 200, false, true, true},
		{"denied", `{}`, 403, true, true, true},
		{"redirect", `{}`, 302, true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := &ArchivedCopilotClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.Path != "/repos/owner/current/copilot/code-review/settings" {
					t.Fatalf("unexpected request %s %s", req.Method, req.URL)
				}
				return labelsResponse(req, tc.status, tc.body), nil
			})).rest}
			selected := settings.ArchivedCopilotToggles{}
			if tc.instructions {
				selected.Instructions = &tc.instructions
			}
			if tc.mcp {
				selected.MCP = &tc.mcp
			}
			got, err := c.Read(repository.Repository{Owner: "owner", Name: "current"}, selected)
			if (err != nil) != tc.fail || calls != 1 {
				t.Fatalf("got=%+v err=%v calls=%d", got, err, calls)
			}
			if !tc.fail && (tc.instructions != (got.Instructions != nil) || tc.mcp != (got.MCP != nil)) {
				t.Fatalf("wrong selected evidence: %+v", got)
			}
		})
	}
}
