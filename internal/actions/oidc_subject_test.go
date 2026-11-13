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

func TestOIDCImmutableReportUnits(t *testing.T) {
	raw := `[{"field":"repository.settings.actions.oidc.subject_claim","before":{"use_immutable_subject_claim":true},"after":{"use_immutable_subject_claim":false}}]`
	var changes []PlanChange
	if err := json.Unmarshal([]byte(raw), &changes); err != nil {
		t.Fatal(err)
	}
	result, err := validateOIDCSubjectReport(changes)
	if err != nil || len(result) != 1 {
		t.Fatal(result, err)
	}
	b, a := result[0].Values()
	if b.UseDefaultTemplate != nil || a.UseDefaultTemplate != nil || b.SubjectClaimTemplate != nil {
		t.Fatal("unselected template reported", b, a)
	}
	for _, bad := range []string{
		strings.Replace(raw, `"use_immutable_subject_claim":true`, `"use_immutable_subject_claim":null`, 1),
		strings.Replace(raw, `"use_immutable_subject_claim":true`, `"use_immutable_subject_claim":true,"use_default_template":true`, 1),
		strings.Replace(raw, `"use_immutable_subject_claim":true`, `"Use_immutable_subject_claim":true`, 1),
		strings.Replace(raw, `"use_immutable_subject_claim":true`, `"use_immutable_subject":true`, 1),
		strings.Replace(raw, `"use_immutable_subject_claim":false`, `"use_immutable_subject_claim":true`, 1),
	} {
		if err := json.Unmarshal([]byte(bad), &changes); err != nil {
			t.Fatal(err)
		}
		if _, err := validateOIDCSubjectReport(changes); err == nil {
			t.Fatal("invalid report accepted", bad)
		}
	}
}
