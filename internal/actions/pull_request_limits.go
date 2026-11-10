package actions

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

func validatePullRequestLimitsReport(raw []PlanChange) ([]settings.PullRequestLimitsChange, error) {
	if len(raw) > settings.PullRequestLimitsFieldCount {
		return nil, fmt.Errorf("too many pull request limit controls")
	}
	seen := map[string]bool{}
	var result []settings.PullRequestLimitsChange
	for _, entry := range raw {
		if seen[entry.Field] || entry.Normalization != nil || entry.SecurityStatusNormalization != nil {
			return nil, fmt.Errorf("invalid pull request limits report")
		}
		seen[entry.Field] = true
		before, after := entry.Before, entry.After
		if entry.Field == settings.PullRequestLimitMaximumPath {
			var err error
			before, err = pullLimitReportInt(before)
			if err != nil {
				return nil, err
			}
			after, err = pullLimitReportInt(after)
			if err != nil {
				return nil, err
			}
		}
		change, err := settings.NewPullRequestLimitsChange(entry.Field, before, after)
		if err != nil {
			return nil, err
		}
		result = append(result, change)
	}
	return result, nil
}
func pullLimitReportInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case float64:
		if n >= 1 && n <= 1000 && n == float64(int(n)) {
			return int(n), nil
		}
	}
	return 0, fmt.Errorf("invalid pull request maximum report")
}
