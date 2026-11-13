package github

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

var classicTestRepo = repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}

func classicStatusDesired(strict bool, checks ...settings.ClassicStatusCheck) *settings.ClassicStatusChecks {
	yes := true
	values := append([]settings.ClassicStatusCheck{}, checks...)
	return &settings.ClassicStatusChecks{Enabled: &yes, RequireBranchesToBeUpToDateBeforeMerging: &strict, StatusChecks: &values}
}

func TestClassicStatusChecksWritesAndAppGuards(t *testing.T) {
	for _, mode := range []string{"changed", "noop", "strict only", "unrelated flag", "disabled enable", "create", "create empty", "disable", "pre null", "pre wrong type", "pre wrong id", "pre errors", "post null", "post errors", "ack missing", "readback mismatch", "alias mismatch", "omitted drift"} {
		t.Run(mode, func(t *testing.T) {
			rule := classicFixtureRule(7, "main")
			if mode == "disabled enable" {
				rule["requiresStatusChecks"] = false
				rule["requiredStatusCheckContexts"] = nil
				rule["requiredStatusChecks"] = nil
			}
			f := newClassicAPIFixture(t, rule)
			if mode == "create" || mode == "create empty" {
				f.rules = nil
			}
			observed, err := f.client.Read(classicTestRepo)
			if err != nil {
				t.Fatal(err)
			}
			d := settings.ClassicRule{BranchNamePattern: "main", RequireStatusChecksToPassBeforeMerging: classicStatusDesired(false, settings.ClassicStatusCheck{Name: "build", AcceptUpdatesFrom: "APP_new"}, settings.ClassicStatusCheck{Name: "other", AcceptUpdatesFrom: "APP_new"}, settings.ClassicStatusCheck{Name: "legacy", AcceptUpdatesFrom: "any"})}
			if len(observed.Rules) > 0 {
				d.ID = observed.Rules[0].Rule.ID
			}
			if mode == "noop" || mode == "unrelated flag" || mode == "strict only" {
				d.RequireStatusChecksToPassBeforeMerging = classicStatusDesired(mode != "strict only", settings.ClassicStatusCheck{Name: "build", AcceptUpdatesFrom: "APP_private"})
			}
			yes, no := true, false
			if mode == "unrelated flag" {
				d.AllowDeletions = &yes
			}
			if mode == "create empty" {
				d.RequireStatusChecksToPassBeforeMerging = classicStatusDesired(false)
			}
			if mode == "disable" {
				d.RequireStatusChecksToPassBeforeMerging = &settings.ClassicStatusChecks{Enabled: &no}
				f.mutate = func(r map[string]any) {
					r["requiredStatusCheckContexts"] = nil
					r["requiredStatusChecks"] = nil
					r["requiresStrictStatusChecks"] = false
				}
			}
			if mode == "omitted drift" {
				d.RequireStatusChecksToPassBeforeMerging = nil
				d.AllowDeletions = &yes
				f.mutate = func(r map[string]any) {
					r["requiredStatusChecks"] = []any{map[string]any{"context": "build", "app": nil}}
				}
			}
			selections, err := settings.PrepareClassicRules(observed, []settings.ClassicRule{d})
			if err != nil {
				t.Fatal(err)
			}
			f.calls = nil
			f.hook = func(phase string, body, variables map[string]any) {
				post := len(f.writes) > 0
				if phase == "app" {
					switch mode {
					case "pre null":
						body["node"] = nil
					case "pre wrong type":
						body["node"] = map[string]any{"__typename": "User", "id": variables["id"]}
					case "pre wrong id":
						body["node"] = map[string]any{"__typename": "App", "id": "unexpected"}
					case "pre errors":
						f.graphqlErrors = []any{map[string]any{"message": "PRIVATE_APP_ERROR"}}
					case "post null":
						if post {
							body["node"] = nil
						}
					case "post errors":
						if post {
							f.graphqlErrors = []any{map[string]any{"message": "PRIVATE_APP_ERROR"}}
						}
					}
				}
				if phase == "write" && mode == "ack missing" {
					delete(body["updateBranchProtectionRule"].(map[string]any)["branchProtectionRule"].(map[string]any), "requiredStatusCheckContexts")
				}
				if phase == "node" && post {
					node := body["node"].(map[string]any)
					if mode == "readback mismatch" {
						node["requiresStrictStatusChecks"] = true
					}
					if mode == "alias mismatch" {
						node["requiredStatusCheckContexts"] = []string{"different"}
					}
				}
			}
			result, err := f.client.Apply(classicTestRepo, selections[0])
			bad := mode == "pre null" || mode == "pre wrong type" || mode == "pre wrong id" || mode == "pre errors" || mode == "post null" || mode == "post errors" || mode == "ack missing" || mode == "readback mismatch" || mode == "alias mismatch" || mode == "omitted drift"
			if (err != nil) != bad {
				t.Fatal(mode, result, err, f.calls)
			}
			var pre interface{ ReadFailedBeforeWrite() }
			preFailure := mode == "pre null" || mode == "pre wrong type" || mode == "pre wrong id" || mode == "pre errors"
			if preFailure {
				if len(f.writes) != 0 || !errors.As(err, &pre) {
					t.Fatal("prewrite certainty", result, err)
				}
				return
			}
			if mode == "noop" {
				if len(f.writes) != 0 || result.Wrote || !reflect.DeepEqual(f.calls, []string{"node", "actors", "actors", "actors", "actors"}) {
					t.Fatal("noop performed work", f.calls)
				}
				return
			}
			if len(f.writes) != 1 || !result.Wrote || result.ID <= 0 {
				t.Fatal(result, err, f.writes)
			}
			input := f.writes[0]
			if _, ok := input["requiredStatusCheckContexts"]; ok {
				t.Fatal("legacy alias sent")
			}
			if mode == "create empty" {
				if input["requiresStatusChecks"] != true || input["requiresStrictStatusChecks"] != false || len(input["requiredStatusChecks"].([]any)) != 0 {
					t.Fatal(input)
				}
			}
			if mode == "disabled enable" || mode == "create" {
				if input["requiresStatusChecks"] != true || input["requiresStrictStatusChecks"] != false {
					t.Fatal("incomplete activation", input)
				}
			}
			if mode == "strict only" {
				if len(input) != 2 || input["requiresStrictStatusChecks"] != false {
					t.Fatal(input)
				}
			}
			if mode == "disable" {
				if len(input) != 2 || input["requiresStatusChecks"] != false {
					t.Fatal(input)
				}
			}
			apps := 0
			for _, call := range f.calls {
				if call == "app" {
					apps++
				}
			}
			if !bad && mode != "disable" && mode != "create empty" && apps != 2 {
				t.Fatal("App identities not deduplicated or independently verified", f.calls)
			}
			if !bad && (mode == "disable" || mode == "create empty") && apps != 0 {
				t.Fatal("unneeded App lookup", f.calls)
			}
			if err != nil && errors.As(err, &pre) {
				t.Fatal("postwrite misclassified", err)
			}
		})
	}
}

func TestClassicStatusChecksOrderAndInactivePreservation(t *testing.T) {
	for _, mode := range []string{"order-only", "inactive-nullness-drift"} {
		t.Run(mode, func(t *testing.T) {
			rule := classicFixtureRule(7, "main")
			rule["requiredStatusCheckContexts"] = []string{"z", "a"}
			rule["requiredStatusChecks"] = []any{map[string]any{"context": "z", "app": nil}, map[string]any{"context": "a", "app": map[string]any{"id": "APP_a"}}}
			if mode == "inactive-nullness-drift" {
				rule["requiresStatusChecks"] = false
				rule["requiredStatusCheckContexts"] = nil
			}
			f := newClassicAPIFixture(t, rule)
			observed, err := f.client.Read(classicTestRepo)
			if err != nil {
				t.Fatal(err)
			}
			d := settings.ClassicRule{ID: observed.Rules[0].Rule.ID, BranchNamePattern: "main", RequireStatusChecksToPassBeforeMerging: classicStatusDesired(true, settings.ClassicStatusCheck{Name: "a", AcceptUpdatesFrom: "APP_a"}, settings.ClassicStatusCheck{Name: "z", AcceptUpdatesFrom: "any"})}
			if mode == "inactive-nullness-drift" {
				yes := true
				d.RequireStatusChecksToPassBeforeMerging, d.AllowDeletions = nil, &yes
				f.mutate = func(r map[string]any) { r["requiredStatusCheckContexts"] = []string{} }
			}
			selected, err := settings.PrepareClassicRules(observed, []settings.ClassicRule{d})
			if err != nil {
				t.Fatal(err)
			}
			f.calls = nil
			result, err := f.client.Apply(classicTestRepo, selected[0])
			if mode == "order-only" {
				if err != nil || result.Wrote || len(f.writes) != 0 || !reflect.DeepEqual(f.calls, []string{"node", "actors", "actors", "actors", "actors"}) {
					t.Fatal("order-only change wrote or resolved Apps", result, err, f.calls)
				}
			} else {
				var pre interface{ ReadFailedBeforeWrite() }
				if err == nil || !result.Wrote || errors.As(err, &pre) {
					t.Fatal("inactive raw nullness drift was not an uncertain failure", result, err)
				}
			}
		})
	}
}

func TestClassicStatusChecksObservationCompleteness(t *testing.T) {
	for _, mode := range []string{"missing-app", "null-app", "blank-app", "missing-contexts", "active-null-checks", "duplicate-names", "incoherent-alias", "partial-error"} {
		t.Run(mode, func(t *testing.T) {
			rule := classicFixtureRule(7, "main")
			check := map[string]any{"context": "build", "app": map[string]any{"id": "APP_source"}}
			rule["requiredStatusChecks"] = []any{check}
			switch mode {
			case "missing-app":
				delete(check, "app")
			case "null-app":
				check["app"] = nil
			case "blank-app":
				check["app"] = map[string]any{"id": " "}
			case "missing-contexts":
				delete(rule, "requiredStatusCheckContexts")
			case "active-null-checks":
				rule["requiredStatusChecks"] = nil
			case "duplicate-names":
				rule["requiredStatusChecks"] = []any{check, map[string]any{"context": "build", "app": nil}}
			case "incoherent-alias":
				rule["requiredStatusCheckContexts"] = []string{"different"}
			}
			f := newClassicAPIFixture(t, rule)
			if mode == "partial-error" {
				f.graphqlErrors = []any{map[string]any{"message": "partial status data"}}
			}
			observed, err := f.client.Read(classicTestRepo)
			if mode == "null-app" {
				if err != nil || (*observed.Rules[0].Rule.RequireStatusChecksToPassBeforeMerging.StatusChecks)[0].AcceptUpdatesFrom != "any" {
					t.Fatal(observed, err)
				}
			} else if err == nil {
				t.Fatal("unknown/unsupported status observation became readable", mode, observed)
			}
		})
	}
}
