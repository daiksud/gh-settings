package settings

import "testing"

func TestCopilotMCPComparison(t *testing.T) {
	for _, tc := range []struct {
		name             string
		acquired, before bool
		desired          *bool
		managed, changes int
		fail             bool
	}{
		{"omitted unread", false, false, nil, 0, 0, false},
		{"unread false", false, false, mcpBool(false), 0, 0, true},
		{"matching false", true, false, mcpBool(false), 1, 0, false},
		{"matching true", true, true, mcpBool(true), 1, 0, false},
		{"disable", true, true, mcpBool(false), 1, 1, false},
		{"enable", true, false, mcpBool(true), 1, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var observed CopilotMCPObservation
			if tc.acquired {
				observed = NewCopilotMCPObservation(tc.before)
			}
			managed, changes, err := CompareCopilotMCP(observed, tc.desired)
			if (err != nil) != tc.fail || managed != tc.managed || len(changes) != tc.changes {
				t.Fatalf("managed=%d changes=%v err=%v", managed, changes, err)
			}
			if len(changes) > 0 {
				before, after := changes[0].Values()
				if before != tc.before || after != *tc.desired || changes[0].Field() != "repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests" {
					t.Fatal(changes[0])
				}
			}
		})
	}
}
func mcpBool(value bool) *bool { return &value }
