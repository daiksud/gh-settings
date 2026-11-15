//go:build acceptance

package acceptance

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// These local fake responses deliberately contradict themselves. Neither an
// empty preflight nor an empty post-delete inventory may be inferred from them.
func TestVariableInventoryAmbiguityFailsClosed(t *testing.T) {
	const secret = "PRIVATE_VARIABLE_INVENTORY_CANARY"
	for _, family := range []string{"actions", "agents", "environment"} {
		for _, stage := range []string{"preflight", "readback"} {
			for _, ambiguity := range []string{"duplicate", "alias", "next", "prev-only"} {
				t.Run(family+"/"+stage+"/"+ambiguity, func(t *testing.T) {
					var mu sync.Mutex
					lists, reads, deletes, unexpected := 0, 0, 0, 0
					base, scope, config := "/repos/owner/current/"+family+"/variables", family+"-variables", "settings: {"+family+": {variables: []}}\n"
					if family == "environment" {
						base, scope = "/repos/owner/current/environments/production/variables", "environment-variables"
						config = "settings: {environments: [{name: production, variables: []}]}\n"
					}
					fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
						mu.Lock()
						defer mu.Unlock()
						if family == "environment" && r.URL.Path == "/repos/owner/current/environments/production" && r.Method == http.MethodGet {
							fmt.Fprint(w, `{"id":42,"name":"production"}`)
							return
						}
						switch {
						case r.URL.Path == base && r.Method == http.MethodGet:
							lists++
							if stage == "preflight" || deletes > 0 {
								if ambiguity == "duplicate" {
									fmt.Fprintf(w, `{"total_count":1,"total_count":0,"variables":[{"name":"OLD","value":%q}],"variables":[]}`, secret)
								} else if ambiguity == "alias" {
									fmt.Fprintf(w, `{"total_count":1,"TOTAL_COUNT":0,"variables":[{"name":"OLD","value":%q}],"VARIABLES":[]}`, secret)
								} else if ambiguity == "next" {
									w.Header().Add("Link", `<https://example.invalid/next>; rel="next"`)
									fmt.Fprint(w, `{"total_count":0,"variables":[]}`)
								} else {
									w.Header().Add("Link", `<https://example.invalid/prev>; rel="prev"`)
									entries := make([]string, 30)
									for i := range entries {
										entries[i] = fmt.Sprintf(`{"name":"ITEM_%d","value":%q}`, i, secret)
									}
									fmt.Fprint(w, `{"total_count":31,"variables":[`+strings.Join(entries, ",")+`]}`)
								}
								return
							}
							fmt.Fprintf(w, `{"total_count":1,"variables":[{"name":"OLD","value":%q}]}`, secret)
						case r.URL.Path == base+"/OLD" && r.Method == http.MethodGet:
							reads++
							fmt.Fprintf(w, `{"name":"OLD","value":%q}`, secret)
						case r.URL.Path == base+"/OLD" && r.Method == http.MethodDelete:
							deletes++
							w.WriteHeader(http.StatusNoContent)
						default:
							unexpected++
							w.WriteHeader(http.StatusBadRequest)
						}
					}, false)
					if err != nil {
						t.Fatal(err)
					}
					defer fixture.close()
					env := append(exportFixtureEnv(fixture.cfg), "GH_DEBUG=api")
					if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
						t.Fatal(err)
					}
					input := repositoryInput(config)
					if err := writeSettingsFile(fixture.work, input); err != nil {
						t.Fatal(err)
					}
					result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"apply", "--prune", scope}, false)
					if err != nil || result.runErr == nil {
						t.Fatalf("ambiguous inventory accepted: setup=%v run=%v", err, result.runErr)
					}
					combined := result.stdout.String() + result.stderr.String()
					if strings.Contains(combined, secret) {
						t.Fatal("variable value leaked")
					}
					if stage == "readback" && !strings.Contains(combined, "write may have applied") {
						t.Fatal("post-delete uncertainty not reported")
					}
					after, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
					if err != nil || string(after) != input {
						t.Fatalf("configuration changed: %v", err)
					}
					mu.Lock()
					defer mu.Unlock()
					if unexpected != 0 || stage == "preflight" && (lists != 1 || reads != 0 || deletes != 0) || stage == "readback" && (lists < 3 || reads != 1 || deletes != 1) {
						t.Fatalf("fixture did not reach expected boundary: lists=%d reads=%d deletes=%d unexpected=%d", lists, reads, deletes, unexpected)
					}
				})
			}
		}
	}
}

func TestVariableInventoryAmbiguousExportRetainsOnlyFailedFamily(t *testing.T) {
	var mu sync.Mutex
	reads := map[string]int{}
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		reads[r.URL.Path]++
		if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
			return
		}
		if isMergeMethodsFixtureRead(r) {
			fmt.Fprint(w, mergeMethodsFixtureResponse)
			return
		}
		switch r.URL.Path {
		case "/repos/owner/current":
			fmt.Fprint(w, planResponse)
		case "/repos/owner/current/topics":
			fmt.Fprint(w, `{"names":[]}`)
		case "/repos/owner/current/labels":
			fmt.Fprint(w, `[]`)
		case "/repos/owner/current/actions/variables":
			fmt.Fprint(w, `{"total_count":1,"total_count":0,"variables":[]}`)
		case "/repos/owner/current/agents/variables":
			fmt.Fprint(w, `{"total_count":1,"variables":[{"name":"NEW_AGENT","value":"new-agent-value"}]}`)
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}, false, "/actions/variables", "/agents/variables", "/labels", "/topics")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.close()
	env := append(exportFixtureEnv(fixture.cfg), "GH_DEBUG=api")
	if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
		t.Fatal(err)
	}
	input := repositoryInput("settings: {actions: {variables: [{name: OLD_ACTION, value: old-action-value}]}, agents: {variables: [{name: OLD_AGENT, value: old-agent-value}]}}\n")
	if err := writeSettingsFile(fixture.work, input); err != nil {
		t.Fatal(err)
	}
	result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"export"}, false)
	if err != nil || result.runErr != nil {
		t.Fatalf("export setup=%v run=%v stderr=%s", err, result.runErr, result.stderr.String())
	}
	output, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	if !strings.Contains(text, "OLD_ACTION") || !strings.Contains(text, "old-action-value") || !strings.Contains(text, "NEW_AGENT") || !strings.Contains(text, "new-agent-value") || strings.Contains(text, "OLD_AGENT") {
		t.Fatal("export failed to retain only the failed family")
	}
	if strings.Contains(result.stdout.String()+result.stderr.String(), "old-action-value") || strings.Contains(result.stdout.String()+result.stderr.String(), "new-agent-value") {
		t.Fatal("export diagnostics leaked values")
	}
	mu.Lock()
	defer mu.Unlock()
	if reads["/repos/owner/current/actions/variables"] != 1 || reads["/repos/owner/current/agents/variables"] != 1 {
		t.Fatalf("inventory reads: %v", reads)
	}
}
