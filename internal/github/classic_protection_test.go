package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func classicFixtureTarget() map[string]any {
	return map[string]any{"id": "R_current", "nameWithOwner": "owner/current", "isPrivate": false, "isArchived": false, "owner": map[string]any{"__typename": "User"}, "squashMergeAllowed": true, "rebaseMergeAllowed": false}
}
func classicFixtureRule(id int64, pattern string) map[string]any {
	return map[string]any{
		"id": fmt.Sprintf("B_%d", id), "databaseId": id, "pattern": pattern, "repository": classicFixtureTarget(),
		"requiresConversationResolution": false, "requiresCommitSignatures": false, "requiresLinearHistory": false, "allowsDeletions": false,
		"isAdminEnforced": true, "allowsForcePushes": false, "blocksCreations": false, "dismissesStaleReviews": true, "lockBranch": false, "lockAllowsFetchAndMerge": false, "requireLastPushApproval": true, "requiresApprovingReviews": true, "requiresCodeOwnerReviews": true, "requiresDeployments": true, "requiresStatusChecks": true, "requiresStrictStatusChecks": true, "restrictsPushes": false, "restrictsReviewDismissals": false,
		"requiredApprovingReviewCount": 2, "requiredDeploymentEnvironments": []string{"production"}, "requiredStatusCheckContexts": []string{"build"}, "requiredStatusChecks": []any{map[string]any{"context": "build", "app": map[string]any{"id": "APP_private"}}},
	}
}
func classicFixturePage(nodes []any, total int, next bool, cursor any) map[string]any {
	return map[string]any{"totalCount": total, "nodes": nodes, "pageInfo": map[string]any{"hasNextPage": next, "endCursor": cursor}}
}
func classicFixtureAck(rule map[string]any) map[string]any {
	result := map[string]any{}
	for _, key := range []string{"id", "databaseId", "pattern", "requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "isAdminEnforced", "requiresApprovingReviews", "requiredApprovingReviewCount", "dismissesStaleReviews", "requiresCodeOwnerReviews", "requireLastPushApproval", "requiresDeployments", "requiredDeploymentEnvironments", "requiresStatusChecks", "requiresStrictStatusChecks", "requiredStatusCheckContexts", "requiredStatusChecks", "repository"} {
		result[key] = rule[key]
	}
	return result
}
func classicFixtureClone(v map[string]any) map[string]any {
	data, _ := json.Marshal(v)
	var result map[string]any
	_ = json.Unmarshal(data, &result)
	return result
}

type classicAPIFixture struct {
	response      func(string, *http.Request) (*http.Response, error)
	t             *testing.T
	client        *ClassicProtectionClient
	rules         []map[string]any
	calls         []string
	writes        []map[string]any
	hook          func(string, map[string]any, map[string]any)
	mutate        func(map[string]any)
	graphqlErrors []any
	rejectWrites  bool
}

func newClassicAPIFixture(t *testing.T, rules ...map[string]any) *classicAPIFixture {
	f := &classicAPIFixture{t: t, rules: rules}
	f.client = &ClassicProtectionClient{graphql: prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		var request struct {
			Query     string
			Variables map[string]any
		}
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if req.Method != "POST" || req.URL.Path != "/graphql" || strings.Contains(request.Query, "matchingRefs") {
			t.Fatal("wrong endpoint/inventory", request)
		}
		phase := ""
		body := map[string]any{}
		var selected map[string]any
		for _, rule := range f.rules {
			if rule["id"] == request.Variables["id"] {
				selected = classicFixtureClone(rule)
			}
		}
		switch {
		case strings.Contains(request.Query, "ClassicStatusApp"):
			phase = "app"
			body["node"] = map[string]any{"__typename": "App", "id": request.Variables["id"]}
		case strings.Contains(request.Query, "mutation ClassicProtectionDelete"):
			phase = "delete"
			input := request.Variables["input"].(map[string]any)
			f.writes = append(f.writes, input)
			for i, rule := range f.rules {
				if rule["id"] == input["branchProtectionRuleId"] {
					f.rules = append(f.rules[:i], f.rules[i+1:]...)
					break
				}
			}
			body["deleteBranchProtectionRule"] = map[string]any{"__typename": "DeleteBranchProtectionRulePayload"}
		case strings.Contains(request.Query, "mutation ClassicProtectionWrite"):
			if f.rejectWrites {
				f.calls = append(f.calls, "write-denied")
				return labelsResponse(req, 403, `{"message":"denied"}`), nil
			}
			phase = "write"
			input := request.Variables["input"].(map[string]any)
			f.writes = append(f.writes, input)
			name := "updateBranchProtectionRule"
			var rule map[string]any
			if input["repositoryId"] != nil {
				name = "createBranchProtectionRule"
				rule = classicFixtureRule(int64(100+len(f.rules)), input["pattern"].(string))
				f.rules = append(f.rules, rule)
			} else {
				for _, v := range f.rules {
					if v["id"] == input["branchProtectionRuleId"] {
						rule = v
					}
				}
			}
			if rule == nil {
				t.Fatal("unknown rule mutation", input)
			}
			for k, v := range input {
				if k != "repositoryId" && k != "branchProtectionRuleId" {
					rule[k] = v
				}
			}
			if checks, ok := input["requiredStatusChecks"].([]any); ok {
				contexts, values := []string{}, []any{}
				for _, item := range checks {
					check := item.(map[string]any)
					contexts = append(contexts, check["context"].(string))
					var app any
					if check["appId"] != "any" {
						app = map[string]any{"id": check["appId"]}
					}
					values = append(values, map[string]any{"context": check["context"], "app": app})
				}
				rule["requiredStatusCheckContexts"], rule["requiredStatusChecks"] = contexts, values
			}
			if f.mutate != nil {
				f.mutate(rule)
			}
			body[name] = map[string]any{"branchProtectionRule": classicFixtureAck(rule)}
		case strings.Contains(request.Query, "ClassicDeploymentEnvironments"):
			phase = "environments"
			target := classicFixtureTarget()
			target["environments"] = classicFixturePage([]any{map[string]any{"id": "E_prod", "databaseId": nil, "name": "production"}, map[string]any{"id": "E_stage", "databaseId": 2, "name": "staging"}}, 2, false, nil)
			body["repository"] = target
		case strings.Contains(request.Query, "ClassicProtectionInventory"):
			phase = "inventory"
			if request.Variables["owner"] != "owner" || request.Variables["name"] != "current" || !strings.Contains(request.Query, "followRenames:false") {
				t.Fatal(request)
			}
			target := classicFixtureTarget()
			nodes := []any{}
			for _, rule := range f.rules {
				nodes = append(nodes, classicFixtureClone(rule))
			}
			target["branchProtectionRules"] = classicFixturePage(nodes, len(nodes), false, nil)
			body["repository"] = target
		case strings.Contains(request.Query, "ClassicProtectionActors"):
			phase = "actors"
			if selected == nil {
				body["node"] = nil
				break
			}
			group := ""
			for _, key := range []string{"pushAllowances", "reviewDismissalAllowances", "bypassForcePushAllowances", "bypassPullRequestAllowances"} {
				if strings.Contains(request.Query, key+"(") {
					group = key
				}
			}
			if group == "" || !strings.Contains(request.Query, "... on App{id}") {
				t.Fatal(request.Query)
			}
			body["node"] = map[string]any{"__typename": "BranchProtectionRule", "id": selected["id"], "databaseId": selected["databaseId"], "repository": selected["repository"], group: classicFixturePage([]any{}, 0, false, nil)}
		case strings.Contains(request.Query, "ClassicProtectionRule"):
			phase = "node"
			if selected != nil {
				selected["__typename"] = "BranchProtectionRule"
			}
			body["node"] = selected
		case strings.Contains(request.Query, "ClassicProtectionTarget"):
			phase = "target"
			body["repository"] = classicFixtureTarget()
		default:
			t.Fatal("unexpected GraphQL request", request.Query)
		}
		f.calls = append(f.calls, phase)
		if f.hook != nil {
			f.hook(phase, body, request.Variables)
		}
		if f.response != nil {
			if response, err := f.response(phase, req); response != nil || err != nil {
				return response, err
			}
		}
		envelope := map[string]any{"data": body}
		if f.graphqlErrors != nil {
			envelope["errors"] = f.graphqlErrors
		}
		data, _ := json.Marshal(envelope)
		return labelsResponse(req, 200, string(data)), nil
	})).graphql}
	return f
}
func classicRepo() repository.Repository {
	return repository.Repository{Owner: "owner", Name: "current"}
}
func classicSelect(t *testing.T, f *classicAPIFixture, desired settings.ClassicRule) settings.ClassicSelection {
	t.Helper()
	o, err := f.client.Read(classicRepo())
	if err != nil {
		t.Fatal(err)
	}
	selected, err := settings.PrepareClassicRules(o, []settings.ClassicRule{desired})
	if err != nil {
		t.Fatal(err)
	}
	return selected[0]
}
func TestClassicPartialWritesNoopAndReadback(t *testing.T) {
	for index, key := range []string{"requiresConversationResolution", "requiresCommitSignatures", "requiresLinearHistory", "allowsDeletions", "lockBranch", "isAdminEnforced"} {
		t.Run(key, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "release/**"))
			f.rules[0][key] = false
			id, v := int64(7), true
			d := settings.ClassicRule{ID: &id, BranchNamePattern: "release/**"}
			switch index {
			case 0:
				d.RequireConversationResolutionBeforeMerging = &v
			case 1:
				d.RequireSignedCommits = &v
			case 2:
				d.RequireLinearHistory = &v
			case 3:
				d.AllowDeletions = &v
			case 4:
				d.LockBranch = &v
			case 5:
				d.DoNotAllowBypassingTheAboveSettings = &v
			}
			selection := classicSelect(t, f, d)
			f.calls = nil
			result, err := f.client.Apply(classicRepo(), selection)
			if err != nil || !result.Wrote || len(f.writes) != 1 {
				t.Fatal(result, f.calls, err)
			}
			if !reflect.DeepEqual(f.writes[0], map[string]any{"branchProtectionRuleId": "B_7", key: true}) {
				t.Fatal(f.writes)
			}
			if strings.Join(f.calls, ",") != "node,actors,actors,actors,actors,write,node,actors,actors,actors,actors" {
				t.Fatal(f.calls)
			}
			selection = classicSelect(t, f, d)
			result, err = f.client.Apply(classicRepo(), selection)
			if err != nil || result.Wrote || len(f.writes) != 1 {
				t.Fatal("matching write", result, err)
			}
			v = false
			selection = classicSelect(t, f, d)
			_, err = f.client.Apply(classicRepo(), selection)
			if err != nil || f.writes[1][key] != false {
				t.Fatal(f.writes, err)
			}
		})
	}
}
func TestClassicMutationFailureBoundaries(t *testing.T) {
	for _, mode := range []string{"original-pattern", "repository", "archived", "linear-disabled", "missing-before", "stale-readback", "companion-count", "companion-app", "omitted-control", "nullable-drift", "actor-drift", "missing-readback", "wrong-ack-id"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "release/**"))
			id, v := int64(7), true
			d := settings.ClassicRule{ID: &id, BranchNamePattern: "release/**", RequireLinearHistory: &v}
			selection := classicSelect(t, f, d)
			if mode == "nullable-drift" {
				f.rules[0]["requiresDeployments"] = false
				f.rules[0]["requiredDeploymentEnvironments"] = nil
				selection = classicSelect(t, f, d)
			}
			f.hook = func(phase string, body, vars map[string]any) {
				if phase == "node" {
					node := body["node"].(map[string]any)
					if len(f.writes) == 0 {
						switch mode {
						case "original-pattern":
							node["pattern"] = "other"
						case "repository":
							node["repository"].(map[string]any)["id"] = "R_replaced"
						case "archived":
							node["repository"].(map[string]any)["isArchived"] = true
						case "linear-disabled":
							node["repository"].(map[string]any)["squashMergeAllowed"] = false
						case "missing-before":
							delete(node, "lockBranch")
						}
					}
					if len(f.writes) > 0 {
						switch mode {
						case "stale-readback":
							node["requiresLinearHistory"] = false
						case "companion-count":
							node["requiredApprovingReviewCount"] = 3
						case "companion-app":
							node["requiredStatusChecks"] = []any{map[string]any{"context": "build", "app": map[string]any{"id": "APP_changed"}}}
						case "omitted-control":
							node["allowsDeletions"] = true
						case "nullable-drift":
							node["requiredDeploymentEnvironments"] = []any{}
						case "missing-readback":
							delete(node, "requiresCommitSignatures")
						}
					}
				}
				if phase == "write" && mode == "wrong-ack-id" {
					body["updateBranchProtectionRule"].(map[string]any)["branchProtectionRule"].(map[string]any)["id"] = "B_wrong"
				}
				if phase == "actors" && len(f.writes) > 0 && mode == "actor-drift" {
					node := body["node"].(map[string]any)
					if _, ok := node["pushAllowances"]; ok {
						node["pushAllowances"] = classicFixturePage([]any{map[string]any{"actor": map[string]any{"__typename": "User", "id": "U_new"}}}, 1, false, nil)
					}
				}
			}
			result, err := f.client.Apply(classicRepo(), selection)
			if err == nil {
				t.Fatal("unsafe mutation succeeded", mode)
			}
			prewrite := mode == "original-pattern" || mode == "repository" || mode == "archived" || mode == "linear-disabled" || mode == "missing-before"
			var pre *classicPreWriteFailure
			if errors.As(err, &pre) != prewrite || (len(f.writes) == 0) != prewrite || result.Wrote == prewrite {
				t.Fatal(mode, result, len(f.writes), err)
			}
		})
	}
}
func TestClassicCreationAndRecovery(t *testing.T) {
	for _, mode := range []string{"success", "late-collision", "missing-database-id", "wrong-owner", "readback-denied", "existing-id", "existing-node", "target-replaced"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
			v := false
			d := settings.ClassicRule{BranchNamePattern: "qa/**/*", AllowDeletions: &v}
			selection := classicSelect(t, f, d)
			if mode == "late-collision" {
				f.rules = append(f.rules, classicFixtureRule(8, d.BranchNamePattern))
			}
			f.hook = func(phase string, body, vars map[string]any) {
				if phase == "target" && mode == "target-replaced" {
					body["repository"].(map[string]any)["id"] = "R_other"
				}
				if phase == "write" {
					ack := body["createBranchProtectionRule"].(map[string]any)["branchProtectionRule"].(map[string]any)
					switch mode {
					case "missing-database-id":
						ack["databaseId"] = nil
					case "existing-id":
						ack["databaseId"] = 7
					case "existing-node":
						ack["id"] = "B_7"
					case "wrong-owner":
						ack["repository"].(map[string]any)["nameWithOwner"] = "other/repo"
					}
				}
				if phase == "node" && mode == "readback-denied" {
					body["node"] = nil
				}
			}
			result, err := f.client.Apply(classicRepo(), selection)
			if mode == "success" {
				if err != nil || result.ID != 101 || result.NodeID != "B_101" || !result.Wrote {
					t.Fatal(result, err)
				}
				if !reflect.DeepEqual(f.writes[0], map[string]any{"repositoryId": "R_current", "pattern": "qa/**/*", "allowsDeletions": false}) {
					t.Fatal(f.writes)
				}
			} else if err == nil {
				t.Fatal("creation failure accepted", mode)
			}
			if mode == "missing-database-id" && (result.ID != 0 || result.NodeID != "B_101" || !strings.Contains(err.Error(), "B_101")) {
				t.Fatal("lost recovery identity", result, err)
			}
			if (mode == "late-collision" || mode == "target-replaced") && len(f.writes) > 0 {
				t.Fatal("prewrite failure mutated")
			}
		})
	}
}
func TestClassicReadRejectsIncompleteKnownState(t *testing.T) {
	for _, mode := range []string{"unknown-member", "missing-boolean", "null-boolean", "null-enabled-count", "null-enabled-list", "null-actor", "unknown-actor", "duplicate-actor", "wrong-type", "missing-page", "incomplete-page", "duplicate-rule", "unknown-repository", "null-list-member"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "future/*"))
			f.hook = func(phase string, body, vars map[string]any) {
				if phase == "inventory" {
					repo := body["repository"].(map[string]any)
					page := repo["branchProtectionRules"].(map[string]any)
					nodes := page["nodes"].([]any)
					rule := nodes[0].(map[string]any)
					switch mode {
					case "unknown-member":
						rule["newProtection"] = true
					case "missing-boolean":
						delete(rule, "isAdminEnforced")
					case "null-boolean":
						rule["requiresCommitSignatures"] = nil
					case "null-enabled-count":
						rule["requiredApprovingReviewCount"] = nil
					case "null-enabled-list":
						rule["requiredStatusChecks"] = nil
					case "missing-page":
						delete(page, "pageInfo")
					case "incomplete-page":
						page["totalCount"] = 2
					case "duplicate-rule":
						page["nodes"] = append(nodes, rule)
						page["totalCount"] = 2
					case "unknown-repository":
						rule["repository"].(map[string]any)["nameWithOwner"] = "other/repo"
					case "null-list-member":
						rule["requiredDeploymentEnvironments"] = []any{nil}
					}
				}
				if phase == "actors" {
					node := body["node"].(map[string]any)
					if mode == "wrong-type" {
						node["__typename"] = "Repository"
					}
					if _, ok := node["pushAllowances"]; ok {
						var actor any = map[string]any{"__typename": "User", "id": "U_1"}
						if mode == "null-actor" {
							actor = nil
						}
						if mode == "unknown-actor" {
							actor = map[string]any{"__typename": "FutureActor", "id": "X_1"}
						}
						if mode == "null-actor" || mode == "unknown-actor" || mode == "duplicate-actor" {
							nodes := []any{map[string]any{"actor": actor}}
							if mode == "duplicate-actor" {
								nodes = append(nodes, nodes[0])
							}
							node["pushAllowances"] = classicFixturePage(nodes, len(nodes), false, nil)
						}
					}
				}
			}
			if _, err := f.client.Read(classicRepo()); err == nil {
				t.Fatal("incomplete acquisition accepted", mode)
			}
		})
	}
}
func TestClassicNestedAndInventoryPagination(t *testing.T) {
	for _, mode := range []string{"complete", "repeat-inventory", "repeat-actors", "changed-count"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "qa/*"))
			inventoryPages, actorPages := 0, 0
			f.hook = func(phase string, body, vars map[string]any) {
				if phase == "inventory" {
					inventoryPages++
					nodes := []any{}
					start, end := 1, 101
					if vars["cursor"] != nil {
						start, end = 101, 102
					}
					for i := start; i < end; i++ {
						nodes = append(nodes, classicFixtureRule(int64(i), fmt.Sprintf("future/%d/*", i)))
					}
					next := vars["cursor"] == nil
					cursor := "rule-next"
					if mode == "repeat-inventory" {
						next = true
					}
					body["repository"].(map[string]any)["branchProtectionRules"] = classicFixturePage(nodes, 101, next, cursor)
				}
				if phase == "actors" {
					node := body["node"].(map[string]any)
					if _, ok := node["pushAllowances"]; !ok {
						return
					}
					actorPages++
					nodes := []any{}
					start, end := 0, 100
					if vars["cursor"] != nil {
						start, end = 100, 101
					}
					for i := start; i < end; i++ {
						nodes = append(nodes, map[string]any{"actor": map[string]any{"__typename": "User", "id": fmt.Sprintf("U_%d", i)}})
					}
					next := vars["cursor"] == nil
					if mode == "repeat-actors" {
						next = true
					}
					total := 101
					if mode == "changed-count" && vars["cursor"] != nil {
						total = 102
					}
					node["pushAllowances"] = classicFixturePage(nodes, total, next, "actor-next")
				}
			}
			// Supply all node identities to the independent actor requests; inventory still
			// comes from two literal pages containing a branch pattern with no refs query.
			f.rules = nil
			for i := 1; i <= 101; i++ {
				f.rules = append(f.rules, classicFixtureRule(int64(i), fmt.Sprintf("future/%d/*", i)))
			}
			observed, err := f.client.Read(classicRepo())
			if mode == "complete" {
				if err != nil || len(observed.Rules) != 101 || inventoryPages != 2 || actorPages != 202 || len(observed.Rules[100].Companions.PushActors) != 101 {
					t.Fatal(len(observed.Rules), inventoryPages, actorPages, err)
				}
			} else if err == nil {
				t.Fatal(mode)
			}
		})
	}
}

func TestClassicPartialGraphQLErrorRetainsRecoveryIdentity(t *testing.T) {
	for _, mode := range []string{"both IDs", "node only", "numeric only", "invalid numeric"} {
		t.Run(mode, func(t *testing.T) {
			writes := 0
			client := &ClassicProtectionClient{graphql: prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				var request struct{ Query string }
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				response := map[string]any{}
				switch {
				case strings.Contains(request.Query, "ClassicProtectionInventory"):
					target := classicFixtureTarget()
					target["branchProtectionRules"] = classicFixturePage([]any{}, 0, false, nil)
					response["data"] = map[string]any{"repository": target}
				case strings.Contains(request.Query, "ClassicProtectionTarget"):
					response["data"] = map[string]any{"repository": classicFixtureTarget()}
				case strings.Contains(request.Query, "mutation ClassicProtectionWrite"):
					writes++
					ack := classicFixtureAck(classicFixtureRule(101, "release/**"))
					ack["repository"] = nil
					if mode == "node only" {
						ack["databaseId"] = nil
					}
					if mode == "numeric only" {
						ack["id"] = nil
					}
					if mode == "invalid numeric" {
						ack["databaseId"] = -1
					}
					response["data"] = map[string]any{"createBranchProtectionRule": map[string]any{"branchProtectionRule": ack}}
					response["errors"] = []any{map[string]any{"message": "repository resolver unavailable", "path": []string{"createBranchProtectionRule", "branchProtectionRule", "repository"}}}
				default:
					t.Fatal("GraphQL error triggered unexpected follow-up", request.Query)
				}
				data, _ := json.Marshal(response)
				return labelsResponse(req, 200, string(data)), nil
			})).graphql}
			result, err := client.Apply(classicRepo(), settings.ClassicSelection{RepositoryID: "R_current", Desired: settings.ClassicRule{BranchNamePattern: "release/**"}})
			var pre interface{ ReadFailedBeforeWrite() }
			if err == nil || writes != 1 || result.Wrote || errors.As(err, &pre) || !strings.Contains(err.Error(), "repository resolver unavailable") {
				t.Fatal("partial GraphQL response became confirmed or prewrite", result, err, writes)
			}
			wantID, wantNode := int64(101), "B_101"
			if mode == "node only" || mode == "invalid numeric" {
				wantID = 0
			}
			if mode == "numeric only" {
				wantNode = ""
			}
			if result.ID != wantID || result.NodeID != wantNode || !strings.Contains(err.Error(), "recovery node") {
				t.Fatal("lost independently usable recovery identity", result, err)
			}
		})
	}
}
