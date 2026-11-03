package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"strings"
)

func declareUIRulesetQueue(ui uimodel.MergeQueueRule) (*settings.RuleParameters, error) {
	if ui.MergeMethod == nil || ui.BuildConcurrency == nil || ui.MinimumGroupSize == nil || ui.MaximumGroupSize == nil || ui.WaitTimeToMeetMinimumGroupSizeMinutes == nil || ui.RequireAllQueueEntriesToPassRequiredChecks == nil || ui.StatusCheckTimeoutMinutes == nil {
		return nil, fmt.Errorf("require_merge_queue requires all queue controls")
	}
	if *ui.MergeMethod != uimodel.Merge && *ui.MergeMethod != uimodel.Squash && *ui.MergeMethod != uimodel.Rebase {
		return nil, fmt.Errorf("unsupported merge queue merge_method")
	}
	method := strings.ToUpper(string(*ui.MergeMethod))
	grouping := "HEADGREEN"
	if *ui.RequireAllQueueEntriesToPassRequiredChecks {
		grouping = "ALLGREEN"
	}
	return &settings.RuleParameters{MergeMethod: &method, GroupingStrategy: &grouping, MaxEntriesToBuild: copyUIRulesetValue(ui.BuildConcurrency), MinEntriesToMerge: copyUIRulesetValue(ui.MinimumGroupSize), MaxEntriesToMerge: copyUIRulesetValue(ui.MaximumGroupSize), MinEntriesToMergeWaitMinutes: copyUIRulesetValue(ui.WaitTimeToMeetMinimumGroupSizeMinutes), CheckResponseTimeoutMinutes: copyUIRulesetValue(ui.StatusCheckTimeoutMinutes)}, nil
}

func observeUIRulesetQueue(p *settings.RuleParameters) *uimodel.MergeQueueRule {
	method := uimodel.MergeMethod(strings.ToLower(*p.MergeMethod))
	all := *p.GroupingStrategy == "ALLGREEN"
	return &uimodel.MergeQueueRule{MergeMethod: &method, RequireAllQueueEntriesToPassRequiredChecks: &all, BuildConcurrency: copyUIRulesetValue(p.MaxEntriesToBuild), MinimumGroupSize: copyUIRulesetValue(p.MinEntriesToMerge), MaximumGroupSize: copyUIRulesetValue(p.MaxEntriesToMerge), WaitTimeToMeetMinimumGroupSizeMinutes: copyUIRulesetValue(p.MinEntriesToMergeWaitMinutes), StatusCheckTimeoutMinutes: copyUIRulesetValue(p.CheckResponseTimeoutMinutes)}
}
