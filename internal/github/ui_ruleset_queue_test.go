package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIRulesetQueueChoicesAndOwnership(t *testing.T) {
	for _, method := range []uimodel.MergeMethod{uimodel.Merge, uimodel.Squash, uimodel.Rebase} {
		for _, all := range []bool{false, true} {
			ui := uiRulesetBase()
			queue := &uimodel.MergeQueueRule{MergeMethod: rulesetPtr(method), BuildConcurrency: rulesetPtr(0), MinimumGroupSize: rulesetPtr(0), MaximumGroupSize: rulesetPtr(0), WaitTimeToMeetMinimumGroupSizeMinutes: rulesetPtr(0), RequireAllQueueEntriesToPassRequiredChecks: rulesetPtr(all), StatusCheckTimeoutMinutes: rulesetPtr(60)}
			ui.Rules.RequireMergeQueue = queue
			policy, err := RulesetDeclaration(ui)
			if err != nil {
				t.Fatal(err)
			}
			source := apiRulesetBase()
			source.Rules = policy.Rules
			observed, err := RulesetObservation(source)
			if err != nil || !reflect.DeepEqual(observed.Rules.RequireMergeQueue, queue) {
				t.Fatal(observed, err)
			}
			*observed.Rules.RequireMergeQueue.BuildConcurrency = 10
			if *(*policy.Rules)[0].Parameters.MaxEntriesToBuild != 0 {
				t.Fatal("projection aliases native queue")
			}
			ui.TargetBranches = nil
			ui.TargetTags = &uimodel.TagTargets{}
			if _, err := RulesetDeclaration(ui); err == nil {
				t.Fatal("queue accepted for tags")
			}
		}
	}
}

func TestUIRulesetQueueRequiresExplicitCompleteValidSettings(t *testing.T) {
	for _, mode := range []string{"missing", "unknown-method", "negative-build", "large-build", "negative-min", "inverted", "zero-timeout", "negative-wait", "large-wait", "large-timeout"} {
		t.Run(mode, func(t *testing.T) {
			ui := uiRulesetBase()
			q := &uimodel.MergeQueueRule{MergeMethod: rulesetPtr(uimodel.Squash), BuildConcurrency: rulesetPtr(1), MinimumGroupSize: rulesetPtr(1), MaximumGroupSize: rulesetPtr(5), WaitTimeToMeetMinimumGroupSizeMinutes: rulesetPtr(5), RequireAllQueueEntriesToPassRequiredChecks: rulesetPtr(true), StatusCheckTimeoutMinutes: rulesetPtr(60)}
			switch mode {
			case "missing":
				q.RequireAllQueueEntriesToPassRequiredChecks = nil
			case "unknown-method":
				q.MergeMethod = rulesetPtr(uimodel.MergeMethod("SQUASH"))
			case "negative-build":
				q.BuildConcurrency = rulesetPtr(-1)
			case "large-build":
				q.BuildConcurrency = rulesetPtr(101)
			case "negative-min":
				q.MinimumGroupSize = rulesetPtr(-1)
			case "inverted":
				q.MinimumGroupSize = rulesetPtr(6)
			case "zero-timeout":
				q.StatusCheckTimeoutMinutes = rulesetPtr(0)
			case "large-wait":
				q.WaitTimeToMeetMinimumGroupSizeMinutes = rulesetPtr(361)
			case "large-timeout":
				q.StatusCheckTimeoutMinutes = rulesetPtr(361)
			case "negative-wait":
				q.WaitTimeToMeetMinimumGroupSizeMinutes = rulesetPtr(-1)
			}
			ui.Rules.RequireMergeQueue = q
			if _, err := RulesetDeclaration(ui); err == nil {
				t.Fatal("invalid queue accepted")
			}
		})
	}
}
