package github

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestClassicForcePushIdentityDiagnosticsStayPrivate(t *testing.T) {
	const private = "A_PRIVATE_UPSTREAM_IDENTITY_FAILURE"
	for _, mode := range []string{"transport", "http", "graphql", "envelope", "node"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := &ClassicProtectionClient{graphql: prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				switch mode {
				case "transport":
					return nil, errors.New(private)
				case "http":
					return labelsResponse(req, 403, `{"message":"`+private+`"}`), nil
				case "graphql":
					return labelsResponse(req, 200, `{"errors":[{"message":"`+private+`"}]}`), nil
				case "envelope":
					return labelsResponse(req, 200, `{"data":{"nodes":[],"`+private+`":true}}`), nil
				default:
					return labelsResponse(req, 200, `{"data":{"nodes":[{"id":"A_PRIVATE","__typename":"App","`+private+`":true}]}}`), nil
				}
			})).graphql}
			err := client.classicForcePushIdentities(desiredForce("specified_actors", settings.ClassicForcePushActor{Type: "app", ID: "A_PRIVATE"}))
			if err == nil || calls != 1 || strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("unbounded identity diagnostic", calls, err)
			}
		})
	}
}

func TestClassicForcePushPaginationAndBoundedIdentityLookup(t *testing.T) {
	f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
	actors := []settings.ClassicActor{}
	desired := []settings.ClassicForcePushActor{}
	for i := 0; i < 101; i++ {
		id := fmt.Sprintf("U_%03d", i)
		actors = append(actors, settings.ClassicActor{Kind: "User", ID: id})
		desired = append(desired, settings.ClassicForcePushActor{Type: "user", ID: id})
	}
	f.forceActors["B_7"] = actors
	f.hook = func(phase string, body, vars map[string]any) {
		if phase != "actors" {
			return
		}
		node := body["node"].(map[string]any)
		if _, ok := node["bypassForcePushAllowances"]; !ok {
			return
		}
		start, end, next, cursor := 0, 100, true, any("force-next")
		if vars["cursor"] != nil {
			if vars["cursor"] != "force-next" {
				t.Fatal(vars)
			}
			start, end, next, cursor = 100, 101, false, nil
		}
		nodes := []any{}
		for _, a := range actors[start:end] {
			nodes = append(nodes, map[string]any{"actor": map[string]any{"__typename": a.Kind, "id": a.ID}})
		}
		node["bypassForcePushAllowances"] = classicFixturePage(nodes, 101, next, cursor)
	}
	yes := true
	id := int64(7)
	d := settings.ClassicRule{ID: &id, BranchNamePattern: "main", RequireSignedCommits: &yes, AllowForcePushes: desiredForce("specified_actors", desired...)}
	selection := classicSelect(t, f, d)
	f.calls = nil
	result, err := f.client.Apply(classicRepo(), selection)
	if err != nil || !result.Wrote {
		t.Fatal(result, err)
	}
	lookups := 0
	for _, call := range f.calls {
		if call == "force-identities" {
			lookups++
		}
	}
	if lookups != 2 || !reflect.DeepEqual(f.writes, []map[string]any{{"branchProtectionRuleId": "B_7", "requiresCommitSignatures": true}}) {
		t.Fatal(f.calls, f.writes)
	}
}

func TestClassicForcePushUnavailableActorEvidence(t *testing.T) {
	for _, mode := range []string{"null", "unknown", "duplicate", "incomplete", "foreign-parent"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
			f.forceActors["B_7"] = []settings.ClassicActor{{Kind: "User", ID: "U_1"}}
			f.hook = func(phase string, body, _ map[string]any) {
				if phase != "actors" {
					return
				}
				node := body["node"].(map[string]any)
				value, ok := node["bypassForcePushAllowances"]
				if !ok {
					return
				}
				page := value.(map[string]any)
				actor := page["nodes"].([]any)[0].(map[string]any)
				switch mode {
				case "null":
					actor["actor"] = nil
				case "unknown":
					actor["actor"].(map[string]any)["__typename"] = "Bot"
				case "duplicate":
					page["nodes"] = []any{actor, actor}
					page["totalCount"] = 2
				case "incomplete":
					page["totalCount"] = 2
				case "foreign-parent":
					node["databaseId"] = 99
				}
			}
			if _, err := f.client.Read(classicRepo()); err == nil {
				t.Fatal("accepted unavailable actors")
			}
			if len(f.writes) > 0 {
				t.Fatal(f.writes)
			}
		})
	}
}

func desiredForce(mode string, actors ...settings.ClassicForcePushActor) *settings.ClassicForcePush {
	enabled := mode != "disabled"
	g := &settings.ClassicForcePush{Enabled: &enabled}
	if enabled {
		g.WhoCanForcePush = &mode
		if mode == "specified_actors" {
			g.Actors = &actors
		}
	}
	return g
}
func TestClassicForcePushReconciliation(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, mode := range []string{"create-user-app", "create-disabled", "create-everyone", "select", "everyone-latent", "disable", "matching", "matching-actors-other-flag", "omitted-team"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
			if strings.HasPrefix(mode, "create") {
				f.rules = nil
			}
			group := desiredForce("specified_actors", settings.ClassicForcePushActor{Type: "user", ID: "U_1"}, settings.ClassicForcePushActor{Type: "app", ID: "A_2"})
			switch mode {
			case "create-disabled", "disable":
				group = desiredForce("disabled")
			case "create-everyone", "everyone-latent":
				group = desiredForce("everyone")
			case "omitted-team":
				group = nil
			}
			if mode == "everyone-latent" || mode == "disable" || mode == "omitted-team" {
				f.forceActors["B_7"] = []settings.ClassicActor{{Kind: "Team", ID: "T_old"}}
			}
			if strings.HasPrefix(mode, "matching") {
				f.forceActors["B_7"] = []settings.ClassicActor{{Kind: "App", ID: "A_2"}, {Kind: "User", ID: "U_1"}}
			}
			observed, err := f.client.Read(repo)
			if err != nil {
				t.Fatal(err)
			}
			d := settings.ClassicRule{BranchNamePattern: "main", AllowForcePushes: group}
			if len(f.rules) > 0 {
				id := int64(7)
				d.ID = &id
			}
			if mode == "matching-actors-other-flag" || mode == "omitted-team" {
				yes := true
				d.RequireSignedCommits = &yes
			}
			selections, err := settings.PrepareClassicRules(observed, []settings.ClassicRule{d})
			if err != nil {
				t.Fatal(err)
			}
			f.calls = nil
			result, err := f.client.Apply(repo, selections[0])
			if err != nil {
				t.Fatal(result, err)
			}
			if mode == "matching" {
				if result.Wrote || len(f.writes) > 0 || containsForceLookup(f.calls) {
					t.Fatal(result, f.calls)
				}
				return
			}
			if !result.Wrote || len(f.writes) != 1 {
				t.Fatal(result, f.writes)
			}
			want := map[string]any{"branchProtectionRuleId": "B_7"}
			if strings.HasPrefix(mode, "create") {
				want = map[string]any{"repositoryId": "R_current", "pattern": "main"}
			}
			switch mode {
			case "create-user-app", "select":
				want["allowsForcePushes"] = false
				want["bypassForcePushActorIds"] = []any{"U_1", "A_2"}
			case "create-disabled", "disable":
				want["allowsForcePushes"] = false
				want["bypassForcePushActorIds"] = []any{}
			case "create-everyone", "everyone-latent":
				want["allowsForcePushes"] = true
			case "matching-actors-other-flag", "omitted-team":
				want["requiresCommitSignatures"] = true
			}
			if !reflect.DeepEqual(f.writes[0], want) {
				t.Fatal(f.writes[0], want)
			}
			if mode == "matching-actors-other-flag" && !containsForceLookup(f.calls) {
				t.Fatal("matching actors skipped needed mutation lookup", f.calls)
			}
			if (mode == "everyone-latent" || mode == "omitted-team") && len(f.forceActors["B_7"]) != 1 {
				t.Fatal("lost Team")
			}
		})
	}
}
func containsForceLookup(calls []string) bool {
	for _, c := range calls {
		if c == "force-identities" {
			return true
		}
	}
	return false
}

func TestClassicForcePushFailureBoundaries(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, mode := range []string{"lookup-null", "lookup-kind", "lookup-duplicate", "matching-actors-lookup-fails", "ack-missing", "ack-null", "ack-wrong", "readback-actors", "everyone-latent-drift", "partial-create"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
			group := desiredForce("specified_actors", settings.ClassicForcePushActor{Type: "user", ID: "U_1"}, settings.ClassicForcePushActor{Type: "app", ID: "A_2"})
			if mode == "everyone-latent-drift" {
				group = desiredForce("everyone")
				f.forceActors["B_7"] = []settings.ClassicActor{{Kind: "Team", ID: "T"}}
			}
			if mode == "partial-create" {
				f.rules = nil
			}
			if mode == "matching-actors-lookup-fails" {
				f.forceActors["B_7"] = []settings.ClassicActor{{Kind: "User", ID: "U_1"}, {Kind: "App", ID: "A_2"}}
			}
			observed, err := f.client.Read(repo)
			if err != nil {
				t.Fatal(err)
			}
			d := settings.ClassicRule{BranchNamePattern: "main", AllowForcePushes: group}
			if len(f.rules) > 0 {
				id := int64(7)
				d.ID = &id
			}
			if mode == "matching-actors-lookup-fails" {
				yes := true
				d.RequireSignedCommits = &yes
			}
			selected, err := settings.PrepareClassicRules(observed, []settings.ClassicRule{d})
			if err != nil {
				t.Fatal(err)
			}
			f.hook = func(phase string, body, vars map[string]any) {
				if phase == "force-identities" {
					switch mode {
					case "lookup-null", "matching-actors-lookup-fails":
						body["nodes"] = []any{nil, nil}
					case "lookup-kind":
						body["nodes"].([]any)[0].(map[string]any)["__typename"] = "Team"
					case "lookup-duplicate":
						body["nodes"].([]any)[1] = body["nodes"].([]any)[0]
					}
				}
				if phase == "write" {
					key := "updateBranchProtectionRule"
					if mode == "partial-create" {
						key = "createBranchProtectionRule"
					}
					ack := body[key].(map[string]any)["branchProtectionRule"].(map[string]any)
					switch mode {
					case "ack-missing":
						delete(ack, "allowsForcePushes")
					case "ack-null":
						ack["allowsForcePushes"] = nil
					case "ack-wrong":
						ack["allowsForcePushes"] = true
					case "readback-actors", "everyone-latent-drift":
						f.forceActors["B_7"] = []settings.ClassicActor{}
					case "partial-create":
						f.graphqlErrors = []any{map[string]any{"message": "resolver failure"}}
					}
				}
			}
			result, err := f.client.Apply(repo, selected[0])
			if err == nil {
				t.Fatal("expected failure", result)
			}
			pre := strings.HasPrefix(mode, "lookup") || mode == "matching-actors-lookup-fails"
			var marker *classicPreWriteFailure
			if errors.As(err, &marker) != pre || (len(f.writes) == 0) != pre {
				t.Fatal(mode, result, f.writes, err)
			}
			if mode == "partial-create" && (result.ID <= 0 || result.NodeID == "" || result.Wrote) {
				t.Fatal(result, err)
			}
		})
	}
}
