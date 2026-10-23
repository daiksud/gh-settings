package actions_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestPropertySummaryNativeUnionAndMissingFields(t *testing.T) {
	r := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Properties: []actions.PropertyPlanChange{{Name: "clear", Type: "string", Before: "", After: nil}, {Name: "tags", Type: "multi_select", Before: []string{"a"}, After: []string{}}}}
	content, err := actions.RenderSummary(r, summaryMetadata())
	if err != nil || !strings.Contains(content, "→ null") || !strings.Contains(content, "→ []") {
		t.Fatal(content, err)
	}
	b, _ := json.Marshal(r)
	var round actions.PlanReport
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatal(err)
	}
	if _, err := actions.RenderSummary(round, summaryMetadata()); err != nil {
		t.Fatal(err)
	}
	for _, c := range []actions.PropertyPlanChange{{Name: "bad.name", Type: "string", Before: "a", After: nil}, {Name: "flag", Type: "true_false", Before: "true", After: false}, {Name: "text", Type: "string", Before: "a", After: []string{}}, {Name: "text", Type: "url", Before: "a", After: nil}, {Name: "text", Type: "string", Before: nil, After: nil}, {Name: "tags", Type: "multi_select", Before: []string{"a"}, After: []string{"b", "b"}}} {
		bad := r
		bad.Properties = []actions.PropertyPlanChange{c}
		if _, err := actions.RenderSummary(bad, summaryMetadata()); err == nil {
			t.Fatal("forged report", c)
		}
	}
	for _, raw := range []string{`{"name":"x","type":"string","after":null}`, `{"name":"x","type":"string","before":"a"}`, `{"name":"x","type":"string","before":"a","after":null,"extra":true}`} {
		var c actions.PropertyPlanChange
		if json.Unmarshal([]byte(raw), &c) == nil {
			t.Fatal("missing/unknown report fields", raw)
		}
	}
	r.Properties = append(r.Properties, r.Properties[0])
	if _, err := actions.RenderSummary(r, summaryMetadata()); err == nil {
		t.Fatal("duplicate comparison")
	}
}
