package settings

import (
	"fmt"
	"slices"
	"strings"
)

// PruneScopes is an explicit deletion selection, independent of declaration scope.
// Its zero value never deletes unlisted objects.
type PruneScopes struct{ labels, rulesets, actionsVariables, agentsVariables, environmentVariables, actionsPolicies, environments, classicProtection, collaborators, autolinks, deployKeys bool }

// PruneScopeNames advertises the same explicit scopes selected by standalone all.
func PruneScopeNames() []string {
	return []string{"labels", "rulesets", "actions-variables", "agents-variables", "environment-variables", "actions-policies", "environments", "collaborators", "branch-protection-rules", "autolinks", "deploy-keys"}
}

func ParsePruneScopes(value string) (PruneScopes, error) {
	var result PruneScopes
	seen := map[string]bool{}
	parts := strings.Split(value, ",")
	for _, part := range parts {
		if part == "" || seen[part] {
			return PruneScopes{}, fmt.Errorf("--prune requires unique nonempty comma-separated scopes")
		}
		seen[part] = true
		if part == "all" {
			if len(parts) != 1 {
				return PruneScopes{}, fmt.Errorf("--prune all must be used alone")
			}
			return ParsePruneScopes(strings.Join(PruneScopeNames(), ","))
		}
		switch part {
		case "autolinks":
			result.autolinks = true
		case "branch-protection-rules":
			result.classicProtection = true
		case "deploy-keys":
			result.deployKeys = true
		case "labels":
			result.labels = true
		case "rulesets":
			result.rulesets = true
		case "actions-variables":
			result.actionsVariables = true
		case "agents-variables":
			result.agentsVariables = true
		case "environment-variables":
			result.environmentVariables = true
		case "actions-policies":
			result.actionsPolicies = true
		case "collaborators":
			result.collaborators = true
		case "environments":
			result.environments = true
		default:
			return PruneScopes{}, fmt.Errorf("unknown --prune scope %q", part)
		}
	}
	return result, nil
}

func (p PruneScopes) Has(scope string) bool {
	return scope == "deploy-keys" && p.deployKeys || scope == "autolinks" && p.autolinks || scope == "collaborators" && p.collaborators || scope == "branch-protection-rules" && p.classicProtection || scope == "environments" && p.environments || scope == "actions-policies" && p.actionsPolicies || scope == "labels" && p.labels || scope == "rulesets" && p.rulesets || scope == "actions-variables" && p.actionsVariables || scope == "agents-variables" && p.agentsVariables || scope == "environment-variables" && p.environmentVariables
}

func (p PruneScopes) Manages(d Declaration) bool {
	return p.deployKeys && d.DeployKeys != nil || p.autolinks && d.Autolinks != nil || p.collaborators && d.Collaborators != nil || p.classicProtection && d.BranchProtectionRules != nil || p.environments && d.Environments != nil || p.actionsPolicies && d.ActionsPolicies != nil || p.labels && d.Labels != nil || p.rulesets && d.Rulesets != nil || p.actionsVariables && d.Variables != nil || p.agentsVariables && d.AgentsVariables != nil || p.environmentVariables && len(d.EnvironmentVariables) > 0
}

// CompareLabelsWithPrune retains normal ID/create-only semantics, then appends
// deletions from the acquired inventory. New objects are never in that inventory.
func CompareLabelsWithPrune(current LabelObservation, desired *LabelDeclarations, prune bool) (Comparison, error) {
	result, err := CompareLabels(current, desired)
	if err != nil || !prune || desired == nil {
		return result, err
	}
	values, err := current.Values()
	if err != nil {
		return Comparison{}, err
	}
	keep := map[int64]bool{}
	for _, declared := range desired.Values() {
		if declared.ID != nil {
			keep[*declared.ID] = true
		}
	}
	slices.SortFunc(values, func(a, b LabelState) int { return strings.Compare(LabelIdentity(a.Name), LabelIdentity(b.Name)) })
	changes := result.Labels.Values()
	for _, value := range values {
		if !keep[value.ID] {
			change, err := NewLabelDelete(value)
			if err != nil {
				return Comparison{}, err
			}
			changes = append(changes, change)
		}
	}
	result.Labels, err = NewLabelChanges(changes)
	result.Managed = 1
	return result, err
}

func ReconcileRulesetsWithPrune(current RulesetObservation, desired *RulesetDeclarations, prune bool) ([]RulesetChange, error) {
	changes, err := ReconcileRulesets(current, desired)
	if err != nil || !prune || desired == nil {
		return changes, err
	}
	values, err := current.Values()
	if err != nil {
		return nil, err
	}
	if len(current.unmanagedIDs) != 0 || current.excludedPush != 0 {
		return nil, fmt.Errorf("cannot prune Rulesets with inherited, Push or other unmanaged inventory objects")
	}
	keep := map[int64]bool{}
	for _, declared := range desired.Values() {
		if declared.ID != nil {
			keep[*declared.ID] = true
		}
	}
	slices.SortFunc(values, func(a, b RulesetState) int { return strings.Compare(a.Policy.Name, b.Policy.Name) })
	for _, value := range values {
		if keep[value.ID] {
			continue
		}
		n := RulesetNormalizations{Enforcement: value.EnforcementNormalization, QualitySeverity: value.QualitySeverityNormalization, RequiredApprovals: value.RequiredApprovalsNormalization, MergeMethods: value.MergeMethodsNormalization, TeamApprovals: value.TeamApprovalsNormalizations, ScanningAlerts: value.ScanningAlertsNormalizations, ScanningSecurityAlerts: value.ScanningSecurityAlertsNormalizations, BypassModes: value.BypassModeNormalizations}
		if !n.IsZero() {
			return nil, fmt.Errorf("cannot prune Ruleset %d with unsupported normalized policy", value.ID)
		}
		prior := cloneRuleset(value.Policy)
		prior.ID = copyRuleValue(&value.ID)
		change, err := NewRulesetChange("delete", value.ID, &prior, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot prune Ruleset %d: %w", value.ID, err)
		}
		changes = append(changes, change)
	}
	slices.SortFunc(changes, func(a, b RulesetChange) int { return strings.Compare(a.Name(), b.Name()) })
	return changes, nil
}
