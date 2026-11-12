package githubactions

import (
	"strings"
	"testing"
)

func TestCompositePruneInputAndOldCLIRejection(t *testing.T) {
	for _, command := range []string{"plan", "apply"} {
		result := runComposite(t, command, "fixture-token", map[string]string{"inputs.prune": "labels,rulesets", "MOCK_EXPECT_PRUNE": "labels,rulesets"})
		if result.status != 0 {
			t.Fatal(command, result.status, result.log)
		}
		old := runComposite(t, command, "fixture-token", map[string]string{"inputs.prune": "labels", "MOCK_OLD_PRUNE_CLI": "1"})
		if old.status == 0 || old.api != "" || strings.Contains(old.calls, "apply") || strings.Contains(old.calls, "plan") {
			t.Fatal(command, old)
		}
	}
	invalid := runComposite(t, "validate", "", map[string]string{"inputs.prune": "labels"})
	if invalid.status == 0 || invalid.downloaded || invalid.api != "" {
		t.Fatal(invalid)
	}
}
