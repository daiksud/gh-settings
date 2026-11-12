package cli

import (
	"bytes"
	"github.com/daiksud/gh-settings/internal/app"
	"testing"
)

func TestPruneCommaSeparatedOptionsAndLocalRejection(t *testing.T) {
	for _, command := range []string{"plan", "apply"} {
		calls := 0
		ops := Operations{Plan: func(o PlanOptions) (app.PlanResult, error) {
			calls++
			if !o.Prune.Has("labels") || !o.Prune.Has("rulesets") {
				t.Fatal(o)
			}
			return app.PlanResult{}, nil
		}, Apply: func(o ApplyOptions) (app.ApplyResult, error) {
			calls++
			if !o.Prune.Has("labels") || !o.Prune.Has("rulesets") {
				t.Fatal(o)
			}
			return app.ApplyResult{}, nil
		}}
		var out, errout bytes.Buffer
		if Run([]string{command, "--prune", "labels,rulesets"}, &out, &errout, "dev", ops) != 0 || calls != 1 {
			t.Fatal(out.String(), errout.String(), calls)
		}
		for _, tail := range [][]string{{"--prune"}, {"--prune", ""}, {"--prune", "labels,"}, {"--prune", "all,labels"}, {"--prune", "all"}, {"--prune", "environments"}, {"--prune", "labels", "--prune", "rulesets"}, {"--prune", "unknown"}} {
			before := calls
			out.Reset()
			errout.Reset()
			args := append([]string{command}, tail...)
			if Run(args, &out, &errout, "dev", ops) == 0 || calls != before {
				t.Fatal(args, calls, errout.String())
			}
		}
	}
}
