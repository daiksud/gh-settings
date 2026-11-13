package actions

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestOIDCReportBoundary(t *testing.T) {
	raw := `[{"field":"repository.settings.actions.oidc.subject_claim","before":{"use_default_template":true},"after":{"use_default_template":false,"subject_claim_template":["repo","context"]}}]`
	var changes []PlanChange
	if json.Unmarshal([]byte(raw), &changes) != nil {
		t.Fatal("fixture")
	}
	result, err := validateOIDCSubjectReport(changes)
	if err != nil || len(result) != 1 {
		t.Fatal(result, err)
	}
	for _, bad := range []string{strings.Replace(raw, `"use_default_template":true`, `"use_default_template":true,"subject_claim_template":null`, 1), strings.Replace(raw, `["repo","context"]`, `["repo","repo"]`, 1), strings.Replace(raw, `"use_default_template":true`, `"use_default_template":true,"immutable":true`, 1), strings.Replace(raw, `"use_default_template":false`, `"use_default_template":true`, 1)} {
		if json.Unmarshal([]byte(bad), &changes) != nil {
			t.Fatal("fixture")
		}
		if _, err := validateOIDCSubjectReport(changes); err == nil {
			t.Fatal("accepted", bad)
		}
	}
	d := true
	c := settings.OIDCSubjectTemplate{UseDefaultTemplate: &d}
	if _, err := settings.NewOIDCSubjectChange(c, c); err == nil {
		t.Fatal("no-op change")
	}
}
