package settings

import (
	"encoding/json"
	"strings"
	"testing"
)

func forceGroup(mode string, actors ...ClassicForcePushActor) *ClassicForcePush {
	yes, no := true, false
	if mode == "disabled" {
		return &ClassicForcePush{Enabled: &no}
	}
	g := &ClassicForcePush{Enabled: &yes, WhoCanForcePush: &mode}
	if mode == "specified_actors" {
		g.Actors = &actors
	}
	return g
}
func forceState(t *testing.T, all bool, actors ...ClassicActor) ClassicState {
	t.Helper()
	s := classicModelState(7, "main")
	s.Companions.AllowsForcePushes = all
	s.Companions.ForcePushActors = append([]ClassicActor{}, actors...)
	var err error
	s.Rule.AllowForcePushes, err = ClassicForcePushProjection(all, s.Companions.ForcePushActors)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestClassicForcePushTruthTableAndTeamBoundary(t *testing.T) {
	for _, tc := range []struct {
		all        bool
		actors     []ClassicActor
		mode       string
		exportable bool
	}{{false, nil, "disabled", true}, {true, nil, "everyone", true}, {false, []ClassicActor{{"User", "U"}, {"App", "A"}}, "specified_actors", true}, {true, []ClassicActor{{"Team", "T"}}, "everyone", true}, {false, []ClassicActor{{"Team", "T"}}, "specified_actors", false}} {
		s := forceState(t, tc.all, tc.actors...)
		o := ClassicObservation{RepositoryID: "R", Rules: []ClassicState{s}}
		if err := o.Validate(); err != nil {
			t.Fatal(err)
		}
		g := s.Rule.AllowForcePushes
		if tc.mode == "disabled" {
			if *g.Enabled {
				t.Fatal(g)
			}
		} else if !*g.Enabled || *g.WhoCanForcePush != tc.mode {
			t.Fatal(g)
		}
		if _, err := ExportClassicRules(o); (err == nil) != tc.exportable {
			t.Fatal(tc, err)
		}
		yes := true
		desired := ClassicRule{ID: s.Rule.ID, BranchNamePattern: "main", RequireSignedCommits: &yes}
		changes, err := CompareClassicRules(o, []ClassicRule{desired})
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		if err = ValidateClassicChanges(changes); err != nil {
			t.Fatal("acquired Team rejected", err)
		}
		after := cloneClassicState(s)
		after.Rule.RequireSignedCommits = &yes
		if err = VerifyClassicPreservation(s, after, desired); err != nil {
			t.Fatal(err)
		}
		prune, err := PrepareClassicPrune(o, nil)
		if err != nil {
			t.Fatal(err)
		}
		candidates, _ := prune.Candidates()
		if len(candidates) != 1 {
			t.Fatal(candidates)
		}
	}
}
func TestClassicForcePushPreservationAndAuthoritativeClear(t *testing.T) {
	before := forceState(t, false, ClassicActor{"Team", "T"})
	yes := true
	desired := ClassicRule{ID: before.Rule.ID, BranchNamePattern: "main", AllowForcePushes: forceGroup("everyone")}
	after := cloneClassicState(before)
	after.Companions.AllowsForcePushes = true
	after.Rule.AllowForcePushes = forceGroup("everyone")
	if err := VerifyClassicPreservation(before, after, desired); err != nil {
		t.Fatal(err)
	}
	after.Companions.ForcePushActors = []ClassicActor{}
	if err := VerifyClassicPreservation(before, after, desired); err == nil {
		t.Fatal("Everyone erased latent Team")
	}
	desired.AllowForcePushes = forceGroup("disabled")
	after.Companions.AllowsForcePushes = false
	after.Rule.AllowForcePushes = forceGroup("disabled")
	if err := VerifyClassicPreservation(before, after, desired); err != nil {
		t.Fatal(err)
	}
	desired.AllowForcePushes = nil
	desired.RequireSignedCommits = &yes
	after.Rule.RequireSignedCommits = &yes
	if err := VerifyClassicPreservation(before, after, desired); err == nil {
		t.Fatal("omission erased actors")
	}
}
func TestClassicForcePushClosedBeforeAndAfter(t *testing.T) {
	s := forceState(t, false, ClassicActor{"Team", "T"})
	before := s.Rule
	yes := true
	after := ClassicRule{ID: s.Rule.ID, BranchNamePattern: "main", RequireSignedCommits: &yes}
	valid := ClassicChange{Before: &before, After: &after}
	data, _ := json.Marshal(valid)
	var decoded ClassicChange
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := ValidateClassicChanges([]ClassicChange{decoded}); err != nil {
		t.Fatal(err)
	}
	after.AllowForcePushes = CloneClassicForcePush(before.AllowForcePushes)
	if err := ValidateClassicChanges([]ClassicChange{{Before: &before, After: &after}}); err == nil {
		t.Fatal("Team allowed in After")
	}
	for _, pair := range [][2]string{{`"type":"team"`, `"type":"Team"`}, {`"type":"team"`, `"Type":"team"`}, {`"id":"T"`, `"id":null`}, {`"id":"T"`, `"id":"T","id":"T"`}, {`"id":"T"`, `"id":"T","login":"private"`}, {`[{"type":"team","id":"T"}]`, `[{"type":"team","id":"T"},{"type":"app","id":"T"}]`}} {
		bad := strings.Replace(string(data), pair[0], pair[1], 1)
		if bad == string(data) {
			t.Fatal("bad test replacement", pair)
		}
		err := json.Unmarshal([]byte(bad), &decoded)
		if err == nil {
			err = ValidateClassicChanges([]ClassicChange{decoded})
		}
		if err == nil {
			t.Fatal("accepted", bad)
		}
	}
}

func TestClassicForcePushPruneSnapshotAndExportOrdering(t *testing.T) {
	state := forceState(t, true, ClassicActor{"Team", "T_latent"})
	prune, err := PrepareClassicPrune(ClassicObservation{RepositoryID: "R", Rules: []ClassicState{state}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	candidates, _ := prune.Candidates()
	state.Companions.ForcePushActors[0].ID = "T_replaced"
	if err := candidates[0].Verify(state, "R", false); err == nil {
		t.Fatal("latent force actor drift ignored or aliased snapshot")
	}
	active := forceState(t, false, ClassicActor{"User", "U"}, ClassicActor{"App", "A"})
	actors := []ClassicForcePushActor{{Type: "user", ID: "U"}, {Type: "app", ID: "A"}}
	active.Rule.AllowForcePushes.Actors = &actors
	exported, err := ExportClassicRules(ClassicObservation{RepositoryID: "R", Rules: []ClassicState{active}})
	if err != nil {
		t.Fatal(err)
	}
	if (*exported[0].AllowForcePushes.Actors)[0].Type != "app" {
		t.Fatal(exported)
	}
	(*exported[0].AllowForcePushes.Actors)[0].ID = "changed"
	if (*active.Rule.AllowForcePushes.Actors)[1].ID != "A" {
		t.Fatal("export aliases input")
	}
}
