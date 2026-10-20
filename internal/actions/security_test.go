package actions_test

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestSecuritySummaryHasNativeClosedValues(t *testing.T) {
	valid := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Security: []actions.PlanChange{{Field: "security.secret_scanning", Before: false, After: true}}}
	text, err := actions.RenderSummary(valid, summaryMetadata())
	if err != nil || !strings.Contains(text, "settings.security.secret_scanning: false → true") {
		t.Fatal(text, err)
	}
	for _, change := range []actions.PlanChange{{Field: "security.code_security", Before: false, After: true}, {Field: "security.secret_scanning", Before: nil, After: true}, {Field: "security.secret_scanning", Before: false, After: "true"}, {Field: "security.secret_scanning", Before: true, After: true}} {
		invalid := valid
		invalid.Security = []actions.PlanChange{change}
		if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
			t.Fatal("forged security report accepted", change)
		}
	}
	invalid := valid
	invalid.Security = append(invalid.Security, invalid.Security[0])
	invalid.Managed = 2
	if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
		t.Fatal("duplicate accepted")
	}
	invalid = valid
	invalid.Managed = 0
	if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
		t.Fatal("undercount accepted")
	}
	invalid = valid
	invalid.Managed = 2
	invalid.Security = []actions.PlanChange{{Field: "security.secret_scanning", Before: true, After: false}, {Field: "security.secret_scanning_push_protection", Before: false, After: true}}
	if _, err := actions.RenderSummary(invalid, summaryMetadata()); err == nil {
		t.Fatal("contradiction accepted")
	}
}
