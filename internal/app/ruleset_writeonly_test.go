package app

import (
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPlanRejectsWriteOnlyDismissalBeforeRemoteWork(t *testing.T) {
	for _, enabled := range []string{"true", "false"} {
		t.Run(enabled, func(t *testing.T) {
			desired := settings.Declaration{Rulesets: appRulesets(t, `[{id: 7, name: basic, enforcement: active, conditions: {ref_name: {include: ['~ALL']}}, rules: [{type: pull_request, parameters: {dismissal_restriction: {enabled: `+enabled+`, allowed_actors: []}}}]}]`)}
			load := func() (settings.Declaration, error) { return desired, nil }
			if _, err := Validate(load); err != nil {
				t.Fatal("write-only declaration failed local validation", err)
			}
			_, err := Plan(load, func() (repository.Repository, error) {
				t.Fatal("unsupported comparison resolved a target")
				return repository.Repository{}, nil
			}, Reader{})
			if err == nil || !strings.Contains(err.Error(), "plan does not support write-only dismissal_restriction") {
				t.Fatal("unsupported plan did not fail explicitly", err)
			}
		})
	}
}
