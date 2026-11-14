package settings

import "testing"

func TestRunnerPruneScopeAndPresence(t *testing.T) {
	for _, selector := range []string{"runners", "labels,runners", "all"} {
		p, err := ParsePruneScopes(selector)
		if err != nil || !p.Has("runners") {
			t.Fatalf("%s must select runners: %v", selector, err)
		}
		empty := []RunnerLabels{}
		if !p.Manages(Declaration{Runners: &empty}) || p.Manages(Declaration{}) {
			t.Fatalf("%s must manage a present, not omitted collection", selector)
		}
	}
}
