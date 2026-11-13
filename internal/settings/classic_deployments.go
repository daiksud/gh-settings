package settings

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

type ClassicDeployments = uimodel.ClassicDeployments

func ClassicDeploymentsSelected(group *ClassicDeployments) bool {
	return group != nil && group.Enabled != nil
}

func validateClassicEnvironmentNames(names []string) error {
	seen := map[string]bool{}
	for _, name := range names {
		if strings.TrimSpace(name) == "" || strings.ContainsFunc(name, unicode.IsControl) || seen[name] {
			return fmt.Errorf("classic deployment environments require unique exact nonblank names without controls")
		}
		seen[name] = true
	}
	return nil
}

func ValidateClassicDeployments(group *ClassicDeployments) error {
	if group == nil {
		return nil
	}
	if group.Enabled == nil {
		if group.Environments != nil {
			return fmt.Errorf("classic deployment environments require enabled")
		}
		return nil
	}
	if !*group.Enabled {
		if group.Environments != nil {
			return fmt.Errorf("disabled classic deployments forbid environments")
		}
		return nil
	}
	if group.Environments == nil || *group.Environments == nil {
		return fmt.Errorf("enabled classic deployments require an explicit environments list")
	}
	return validateClassicEnvironmentNames(*group.Environments)
}

// ClassicDeploymentProjection preserves nullable inactive raw state separately.
func ClassicDeploymentProjection(enabled bool, raw ClassicNullable[[]string]) (*ClassicDeployments, error) {
	if !raw.Present || enabled && raw.Value == nil {
		return nil, fmt.Errorf("incomplete classic deployment state")
	}
	if raw.Value != nil {
		if err := validateClassicEnvironmentNames(*raw.Value); err != nil {
			return nil, err
		}
	}
	group := &ClassicDeployments{Enabled: &enabled}
	if enabled {
		names := append([]string{}, (*raw.Value)...)
		slices.Sort(names)
		group.Environments = &names
	}
	return group, nil
}

func cloneClassicDeployments(group *ClassicDeployments) *ClassicDeployments {
	if group == nil {
		return nil
	}
	cloned := *group
	cloned.Enabled = cloneClassicValue(group.Enabled)
	if group.Environments != nil {
		names := slices.Clone(*group.Environments)
		cloned.Environments = &names
	}
	return &cloned
}

func classicDeploymentsEqual(a, b *ClassicDeployments) bool {
	if !ClassicDeploymentsSelected(a) || !ClassicDeploymentsSelected(b) {
		return false
	}
	if *a.Enabled != *b.Enabled {
		return false
	}
	if !*a.Enabled {
		return true
	}
	if a.Environments == nil || b.Environments == nil {
		return false
	}
	left, right := slices.Clone(*a.Environments), slices.Clone(*b.Environments)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

// CheckClassicDeploymentPruneConflict never silently retains a deletion candidate.
func CheckClassicDeploymentPruneConflict(rules *[]ClassicRule, candidates []EnvironmentPruneCandidate) error {
	if rules == nil {
		return nil
	}
	for _, rule := range *rules {
		group := rule.RequireDeploymentsToSucceedBeforeMerging
		if err := ValidateClassicDeployments(group); err != nil {
			return err
		}
		if !ClassicDeploymentsSelected(group) || !*group.Enabled {
			continue
		}
		for _, name := range *group.Environments {
			for _, candidate := range candidates {
				if EnvironmentIdentity(name) == EnvironmentIdentity(candidate.Name()) {
					return fmt.Errorf("classic deployment requirement conflicts with a selected Environment deletion; retain the Environment or change the requirement")
				}
			}
		}
	}
	return nil
}
