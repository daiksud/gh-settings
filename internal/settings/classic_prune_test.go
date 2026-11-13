package settings

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestClassicPruneFrozenIdentityAndPrivateEvidence(t *testing.T) {
	keep, remove := classicModelState(7, "main"), classicModelState(8, "main")
	remove.NodeID = "B8"
	app := "PRIVATE_APP"
	checks := []ClassicCheck{{Context: "private-check", AppID: &app}}
	remove.Companions.Checks.Value = &checks
	remove.Companions.PushActors = []ClassicActor{{ID: "PRIVATE_ACTOR", Kind: "User"}}
	observed := ClassicObservation{RepositoryID: "R", Rules: []ClassicState{remove, keep}}
	desired := []ClassicRule{{ID: keep.Rule.ID, BranchNamePattern: "main"}}
	prepared, err := PrepareClassicPrune(observed, desired)
	if err != nil {
		t.Fatal(err)
	}
	candidates, _ := prepared.Candidates()
	if len(candidates) != 1 || candidates[0].ID() != 8 {
		t.Fatal(candidates)
	}
	// Acquired evidence is copied before caller-owned observations/reports can change.
	*remove.Rule.AllowDeletions = true
	*remove.Companions.Checks.Value = nil
	remove.Companions.PushActors[0].ID = "changed"
	change, _ := candidates[0].Change()
	*change.Before.ID = 99
	candidates[0].state.NodeID = "changed"
	candidates, _ = prepared.Candidates()
	if candidates[0].ID() != 8 || candidates[0].NodeID() != "B8" || *candidates[0].state.Rule.AllowDeletions || (*candidates[0].state.Companions.Checks.Value)[0].Context != "private-check" || candidates[0].state.Companions.PushActors[0].ID != "PRIVATE_ACTOR" {
		t.Fatal("aliased snapshot", candidates)
	}
	// Two rules with the same pattern are distinguished by their acquired identities.
	fresh := ClassicObservation{RepositoryID: "R", Rules: []ClassicState{cloneClassicState(candidates[0].state), classicModelState(7, "main"), classicModelState(9, "late/**")}}
	if err := prepared.Recheck(fresh); err != nil {
		t.Fatal("new arrival widened candidates", err)
	}
	for _, mutate := range []func(*ClassicObservation){
		func(o *ClassicObservation) { o.Rules = o.Rules[1:] },
		func(o *ClassicObservation) { o.Rules[0].NodeID = "replacement" },
		func(o *ClassicObservation) { *o.Rules[0].Rule.ID = 90 },
		func(o *ClassicObservation) { o.Rules[0].Rule.BranchNamePattern = "renamed" },
		func(o *ClassicObservation) { *o.Rules[0].Rule.LockBranch = true },
		func(o *ClassicObservation) { *o.Rules[0].Rule.DoNotAllowBypassingTheAboveSettings = true },
		func(o *ClassicObservation) { o.Rules[0].Companions.Contexts.Value = nil },
		func(o *ClassicObservation) { o.Rules[0].Companions.PushActors[0].ID = "other" },
		func(o *ClassicObservation) { *(*o.Rules[0].Companions.Checks.Value)[0].AppID = "other" },
		func(o *ClassicObservation) { o.Archived = true },
		func(o *ClassicObservation) { o.RepositoryID = "other" },
	} {
		next := fresh
		next.Rules = append([]ClassicState{}, fresh.Rules...)
		for i := range next.Rules {
			next.Rules[i] = cloneClassicState(next.Rules[i])
		}
		mutate(&next)
		if prepared.Recheck(next) == nil {
			t.Fatal("candidate drift accepted", next)
		}
	}
	empty, err := PrepareClassicPrune(ClassicObservation{RepositoryID: "R", Rules: []ClassicState{}}, []ClassicRule{})
	if err != nil || empty.Recheck(fresh) != nil {
		t.Fatal("empty selection adopted new arrivals", err)
	}
}

func TestClassicPruneScopePresenceArchiveAndReport(t *testing.T) {
	all, err := ParsePruneScopes("all")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range PruneScopeNames() {
		if !all.Has(name) {
			t.Fatal("all omitted", name)
		}
	}
	empty := []ClassicRule{}
	if all.Manages(Declaration{}) || !all.Manages(Declaration{BranchProtectionRules: &empty}) {
		t.Fatal("collection presence lost")
	}
	observed := ClassicObservation{RepositoryID: "R", Archived: true, Rules: []ClassicState{classicModelState(7, "main")}}
	prepared, err := PrepareClassicPrune(observed, empty)
	if err != nil || prepared.ValidateApply(false) == nil || prepared.ValidateApply(true) != nil {
		t.Fatal(prepared, err)
	}
	observed.Archived = false
	if prepared.Recheck(observed) == nil || prepared.AfterUnarchive().Recheck(observed) != nil {
		t.Fatal("unarchive exception is not exact")
	}
	changes, err := CompareClassicRulesWithPrune(observed, empty, true)
	if err != nil || len(changes) != 1 || changes[0].After != nil || ValidateClassicChanges(changes) != nil {
		t.Fatal(changes, err)
	}
	data, _ := json.Marshal(changes[0])
	var roundtrip ClassicChange
	if err := json.Unmarshal(data, &roundtrip); err != nil || !reflect.DeepEqual(changes[0], roundtrip) {
		t.Fatal(string(data), err)
	}
	for _, raw := range []string{`{"before":null}`, `{"after":null}`, `{"before":{},"raw":{}}`} {
		if json.Unmarshal([]byte(raw), &roundtrip) == nil {
			t.Fatal("invalid deletion decoded", raw)
		}
	}
	incomplete := CloneClassicRule(*changes[0].Before)
	incomplete.AllowDeletions = nil
	for _, bad := range [][]ClassicChange{{{}}, {{Before: &incomplete}}, {changes[0], changes[0]}, {{Before: changes[0].Before}, {After: &ClassicRule{BranchNamePattern: "main"}}}} {
		if ValidateClassicChanges(bad) == nil {
			t.Fatal("invalid deletion accepted", bad)
		}
	}
	second := CloneClassicRule(*changes[0].Before)
	id := int64(8)
	second.ID = &id
	if ValidateClassicChanges(append(changes, ClassicChange{Before: &second})) != nil {
		t.Fatal("distinct identity duplicate patterns rejected")
	}
}
