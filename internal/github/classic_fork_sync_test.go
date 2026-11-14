package github

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func setClassicForkEvidence(value any, fork any) {
	switch v := value.(type) {
	case map[string]any:
		if _, ok := v["isFork"]; ok {
			v["isFork"] = fork
		}
		for _, child := range v {
			setClassicForkEvidence(child, fork)
		}
	case []any:
		for _, child := range v {
			setClassicForkEvidence(child, fork)
		}
	}
}

func TestClassicForkSyncReconciliation(t *testing.T) {
	for _, mode := range []string{"create-true", "create-false", "update-true", "update-false", "matching-true", "matching-false", "lock-with-latent-match", "unlock-preserves", "lock-only"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
			f.hook = func(_ string, body, _ map[string]any) { setClassicForkEvidence(body, true) }
			want := strings.HasSuffix(mode, "true") || mode == "lock-with-latent-match"
			yes, no := true, false
			d := settings.ClassicRule{BranchNamePattern: "main", LockBranch: &yes, AllowForkSyncing: &want}
			if strings.HasPrefix(mode, "create") {
				f.rules = nil
			} else {
				id := int64(7)
				d.ID = &id
				f.rules[0]["lockBranch"] = true
				f.rules[0]["lockAllowsFetchAndMerge"] = !want
				if strings.HasPrefix(mode, "matching") {
					f.rules[0]["lockAllowsFetchAndMerge"] = want
				}
			}
			if mode == "lock-with-latent-match" || mode == "lock-only" {
				f.rules[0]["lockBranch"] = false
				f.rules[0]["lockAllowsFetchAndMerge"] = true
			}
			if mode == "unlock-preserves" {
				d.LockBranch = &no
				d.AllowForkSyncing = nil
				f.rules[0]["lockAllowsFetchAndMerge"] = true
			}
			if mode == "lock-only" {
				d.AllowForkSyncing = nil
			}
			selected := classicSelect(t, f, d)
			result, err := f.client.Apply(classicRepo(), selected)
			if err != nil {
				t.Fatal(result, err)
			}
			if strings.HasPrefix(mode, "matching") {
				if result.Wrote || len(f.writes) != 0 {
					t.Fatal(result, f.writes)
				}
				return
			}
			payload := map[string]any{"branchProtectionRuleId": "B_7", "lockAllowsFetchAndMerge": want}
			if strings.HasPrefix(mode, "create") {
				payload = map[string]any{"repositoryId": "R_current", "pattern": "main", "lockBranch": true, "lockAllowsFetchAndMerge": want}
			}
			if mode == "unlock-preserves" || mode == "lock-with-latent-match" || mode == "lock-only" {
				payload = map[string]any{"branchProtectionRuleId": "B_7", "lockBranch": *d.LockBranch}
			}
			if !result.Wrote || !reflect.DeepEqual(f.writes, []map[string]any{payload}) {
				t.Fatal(result, f.writes, payload)
			}
		})
	}
}

func TestClassicForkSyncTargetAndReadbackFailures(t *testing.T) {
	for _, mode := range []string{"nonfork", "prewrite-fork-drift", "actor-fork-drift", "create-target-drift", "ack-missing-sync", "ack-missing-fork", "ack-fork-drift", "readback-fork-drift", "readback-value-drift", "omitted-drift", "partial-create"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
			f.rules[0]["lockBranch"] = true
			f.hook = func(_ string, body, _ map[string]any) { setClassicForkEvidence(body, mode != "nonfork") }
			yes := true
			d := settings.ClassicRule{BranchNamePattern: "main", LockBranch: &yes, AllowForkSyncing: &yes}
			if mode == "partial-create" || mode == "create-target-drift" {
				f.rules = nil
			} else {
				id := int64(7)
				d.ID = &id
			}
			if mode == "omitted-drift" {
				d.AllowForkSyncing = nil
				d.RequireSignedCommits = &yes
			}
			o, err := f.client.Read(classicRepo())
			if err != nil {
				t.Fatal(err)
			}
			selections, err := settings.PrepareClassicRules(o, []settings.ClassicRule{d})
			if mode == "nonfork" {
				if err == nil {
					t.Fatal("selected nonfork accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			f.hook = func(phase string, body, _ map[string]any) {
				fork := true
				if mode == "prewrite-fork-drift" || mode == "actor-fork-drift" && phase == "actors" || mode == "create-target-drift" && phase == "target" || mode == "readback-fork-drift" && phase == "node" && len(f.writes) > 0 {
					fork = false
				}
				setClassicForkEvidence(body, fork)
				if phase == "write" {
					key := "updateBranchProtectionRule"
					if mode == "partial-create" {
						key = "createBranchProtectionRule"
					}
					ack := body[key].(map[string]any)["branchProtectionRule"].(map[string]any)
					switch mode {
					case "ack-missing-sync":
						delete(ack, "lockAllowsFetchAndMerge")
					case "ack-missing-fork":
						delete(ack["repository"].(map[string]any), "isFork")
					case "ack-fork-drift":
						setClassicForkEvidence(ack, false)
					case "readback-value-drift":
						f.rules[0]["lockAllowsFetchAndMerge"] = false
					case "omitted-drift":
						f.rules[0]["lockAllowsFetchAndMerge"] = true
					case "partial-create":
						f.graphqlErrors = []any{map[string]any{"message": "resolver failed"}}
					}
				}
			}
			result, err := f.client.Apply(classicRepo(), selections[0])
			pre := mode == "prewrite-fork-drift" || mode == "actor-fork-drift" || mode == "create-target-drift"
			var marker *classicPreWriteFailure
			if err == nil || errors.As(err, &marker) != pre || (len(f.writes) == 0) != pre {
				t.Fatal(result, err, f.writes)
			}
			if mode == "partial-create" && (result.ID <= 0 || result.NodeID == "" || result.Wrote) {
				t.Fatal(result, err)
			}
		})
	}
}

func TestClassicForkSyncRequiredTypedForkEvidence(t *testing.T) {
	for _, evidence := range []any{nil, "true", 1} {
		f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
		f.hook = func(_ string, body, _ map[string]any) { setClassicForkEvidence(body, evidence) }
		if _, err := f.client.Read(classicRepo()); err == nil {
			t.Fatal("invalid fork evidence accepted", evidence)
		}
	}
	f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
	f.hook = func(phase string, body, _ map[string]any) {
		if phase == "inventory" {
			delete(body["repository"].(map[string]any), "isFork")
		}
	}
	if _, err := f.client.Read(classicRepo()); err == nil {
		t.Fatal("missing fork evidence accepted")
	}
}

func TestClassicForkSyncInventoryPageAndPruneTargetDrift(t *testing.T) {
	f := newClassicAPIFixture(t, classicFixtureRule(7, "main"), classicFixtureRule(8, "other"))
	f.hook = func(phase string, body, vars map[string]any) {
		setClassicForkEvidence(body, true)
		if phase != "inventory" {
			return
		}
		target := body["repository"].(map[string]any)
		if vars["cursor"] == nil {
			target["branchProtectionRules"] = classicFixturePage([]any{classicFixtureClone(f.rules[0])}, 2, true, "next")
		} else {
			target["branchProtectionRules"] = classicFixturePage([]any{classicFixtureClone(f.rules[1])}, 2, false, nil)
		}
		setClassicForkEvidence(body, vars["cursor"] == nil)
	}
	if _, err := f.client.Read(classicRepo()); err == nil {
		t.Fatal("fork status changed across inventory pages")
	}
	for _, post := range []bool{false, true} {
		f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
		f.hook = func(_ string, body, _ map[string]any) { setClassicForkEvidence(body, true) }
		candidate := classicPruneSelect(t, f)
		f.hook = func(phase string, body, _ map[string]any) { setClassicForkEvidence(body, post && phase != "inventory") }
		err := f.client.Delete(classicRepo(), candidate)
		var marker *classicPreWriteFailure
		if err == nil || errors.As(err, &marker) == post || len(f.writes) != map[bool]int{true: 1, false: 0}[post] {
			t.Fatal(err, f.writes)
		}
		if post && !strings.Contains(err.Error(), "write may have applied") {
			t.Fatal(err)
		}
	}
}
