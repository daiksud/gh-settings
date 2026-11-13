package github

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestClassicDeploymentReferenceDiagnostics(t *testing.T) {
	const canary = "A_PRIVATE_DEPLOYMENT_UPSTREAM_FAILURE"
	for _, afterWrite := range []bool{false, true} {
		for _, failure := range []string{"graphql", "http", "transport", "top member", "repository member", "page member", "pageInfo member", "node member", "node type", "numeric identity"} {
			t.Run(failure+map[bool]string{false: " before", true: " after"}[afterWrite], func(t *testing.T) {
				rule := classicFixtureRule(7, "main")
				rule["requiresDeployments"] = false
				f := newClassicAPIFixture(t, rule)
				id := int64(7)
				selection := classicSelect(t, f, settings.ClassicRule{ID: &id, BranchNamePattern: "main", RequireDeploymentsToSucceedBeforeMerging: deploymentGroup(true, "production")})
				triggered := 0
				f.hook = func(phase string, body map[string]any, _ map[string]any) {
					if phase != "environments" || (len(f.writes) > 0) != afterWrite {
						return
					}
					triggered++
					repo := body["repository"].(map[string]any)
					page := repo["environments"].(map[string]any)
					nodes := page["nodes"].([]any)
					switch failure {
					case "graphql":
						f.graphqlErrors = []any{map[string]any{"message": canary}}
					case "top member":
						body[canary] = true
					case "repository member":
						repo[canary] = true
					case "page member":
						page[canary] = true
					case "pageInfo member":
						page["pageInfo"].(map[string]any)[canary] = true
					case "node member":
						nodes[0].(map[string]any)[canary] = true
					case "node type":
						nodes[0] = canary
					case "numeric identity":
						nodes[0].(map[string]any)["databaseId"] = canary
					}
				}
				f.response = func(phase string, req *http.Request) (*http.Response, error) {
					if phase != "environments" || (len(f.writes) > 0) != afterWrite {
						return nil, nil
					}
					switch failure {
					case "http":
						return labelsResponse(req, 403, `{"message":"`+canary+`"}`), nil
					case "transport":
						return nil, errors.New(canary)
					}
					return nil, nil
				}
				result, err := f.client.Apply(classicRepo(), selection)
				var pre *classicPreWriteFailure
				wantWrites := 0
				if afterWrite {
					wantWrites = 1
				}
				if err == nil || strings.Contains(err.Error(), canary) || triggered != 1 || len(f.writes) != wantWrites || errors.As(err, &pre) == afterWrite || result.Wrote != afterWrite {
					t.Fatal("diagnostic/certainty boundary", result, err, triggered, f.writes)
				}
				reason := map[string]string{
					"graphql": "read classic deployment references: request failed", "http": "read classic deployment references: request failed", "transport": "read classic deployment references: request failed",
					"top member": "invalid classic Environment repository response", "repository member": "incomplete or unsupported classic Environment repository identity",
					"page member": "incomplete classic Environment inventory page", "pageInfo member": "incomplete classic Environment inventory page",
					"node member": "incomplete classic Environment identity object", "node type": "incomplete classic Environment identity object", "numeric identity": "invalid classic Environment numeric identity",
				}[failure]
				if !strings.Contains(err.Error(), reason) {
					t.Fatal("bounded reason lost", err)
				}
				if afterWrite && !strings.Contains(err.Error(), "effects may have applied") {
					t.Fatal("uncertainty marker lost", err)
				}
			})
		}
	}
}
