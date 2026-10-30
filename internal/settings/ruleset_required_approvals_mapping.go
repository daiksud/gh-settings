package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const RulesetRequiredApprovalsPath = "repository.settings.rulesets.rules.require_a_pull_request_before_merging.required_approvals"
const DefaultRulesetRequiredApprovals = uimodel.DefaultRequiredApprovals

func RulesetRequiredApprovalsSupported(value int) bool { return value >= 0 && value <= 10 }
