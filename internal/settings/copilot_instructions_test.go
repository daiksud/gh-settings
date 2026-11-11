package settings

import "testing"

func TestCopilotInstructionsComparison(t *testing.T) {
	for _, tc := range []struct {
		name             string
		acquired, before bool
		desired          *bool
		managed, changes int
		fail             bool
	}{
		{"omitted unread", false, false, nil, 0, 0, false},
		{"unread false", false, false, instructionsBool(false), 0, 0, true},
		{"matching false", true, false, instructionsBool(false), 1, 0, false},
		{"matching true", true, true, instructionsBool(true), 1, 0, false},
		{"disable", true, true, instructionsBool(false), 1, 1, false},
		{"enable", true, false, instructionsBool(true), 1, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var observed CopilotInstructionsObservation
			if tc.acquired {
				observed = NewCopilotInstructionsObservation(tc.before)
			}
			managed, changes, err := CompareCopilotInstructions(observed, tc.desired)
			if (err != nil) != tc.fail || managed != tc.managed || len(changes) != tc.changes {
				t.Fatalf("managed=%d changes=%v err=%v", managed, changes, err)
			}
			if len(changes) > 0 {
				before, after := changes[0].Values()
				if before != tc.before || after != *tc.desired || changes[0].Field() != "repository.settings.copilot.code_review.use_custom_instructions" {
					t.Fatal(changes[0])
				}
			}
		})
	}
}
func instructionsBool(value bool) *bool { return &value }
