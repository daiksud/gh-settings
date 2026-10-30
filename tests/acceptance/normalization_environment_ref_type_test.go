//go:build acceptance

package acceptance

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestEnvironmentRefTypeNormalizationExport(t *testing.T) {
	for _, mode := range []string{"unique", "collision", "missing type", "null type", "latent"} {
		t.Run(mode, func(t *testing.T) {
			fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if servePRAccessFixtureRead(w, r) || serveIssueCreationPolicyFixtureRead(w, r) || serveSponsorshipsFixtureRead(w, r) {
					return
				}
				if isMergeMethodsFixtureRead(r) {
					fmt.Fprint(w, mergeMethodsFixtureResponse)
					return
				}
				if r.Method != "GET" {
					t.Error("export wrote", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				base := "/repos/owner/current/environments/production"
				responses := map[string]string{
					"/repos/owner/current":                planResponse,
					"/repos/owner/current/topics":         `{"names":[]}`,
					"/repos/owner/current/labels":         `[]`,
					"/repos/owner/current/environments":   `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`,
					base:                                  `{"id":1,"name":"production","protection_rules":[],"can_admins_bypass":false,"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}`,
					base + "/deployment_protection_rules": `{"total_count":0,"custom_deployment_protection_rules":[]}`,
					base + "/deployment-branch-policies":  `{"total_count":2,"branch_policies":[{"id":7,"type":"future","name":"main"},{"id":8,"type":"tag","name":"main"}]}`,
				}
				body, ok := responses[r.URL.Path]
				if !ok {
					t.Error("unexpected request", r.Method, r.URL)
					w.WriteHeader(400)
					return
				}
				if mode == "collision" && r.URL.Path == base+"/deployment-branch-policies" {
					body = `{"total_count":2,"branch_policies":[{"id":7,"type":"future","name":"main"},{"id":8,"type":"branch","name":"main"}]}`
				}
				if (mode == "missing type" || mode == "null type") && r.URL.Path == base+"/deployment-branch-policies" {
					body = `{"total_count":2,"branch_policies":[{"id":7,"type":"future","name":"main"},{"id":8,"name":"second"}]}`
					if mode == "null type" {
						body = strings.Replace(body, `"name":"second"`, `"type":null,"name":"second"`, 1)
					}
				}
				if mode == "latent" && r.URL.Path == base {
					body = strings.Replace(body, `"protected_branches":false`, `"protected_branches":true`, 1)
				}
				fmt.Fprint(w, body)
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.close()
			env := exportFixtureEnv(fixture.cfg)
			if err := setupOrdinaryOrigin(fixture.work, env); err != nil {
				t.Fatal(err)
			}
			const existing = "existing destination bytes\n"
			if err := writeSettingsFile(fixture.work, existing); err != nil {
				t.Fatal(err)
			}
			result, err := executeCLI(acceptanceBinary(t), fixture.work, fixture.root, env, []string{"export", "--environments"}, false)
			warning := `Warning: repository.settings.environments.deployment_branch_or_tag_rules.ref_type: observed "future" is outside UI choices; using "branch" (projection only). Environment "production", pattern 7 name_pattern "main".` + "\n"
			if mode == "missing type" || mode == "null type" {
				var failed *exec.ExitError
				if err != nil || !errors.As(result.runErr, &failed) || failed.ExitCode() != 1 || result.stdout.Len() != 0 || strings.Contains(result.stderr.String(), "Warning:") || !strings.Contains(result.stderr.String(), "missing/null type") {
					t.Fatal("incomplete acquisition became normalized success", err, result.runErr, result.stdout.String(), result.stderr.String())
				}
				data, readErr := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
				if readErr != nil || string(data) != existing {
					t.Fatal("failed acquisition replaced destination", string(data), readErr)
				}
				if err := fixture.checkRequests(false); err != nil {
					t.Fatal(err)
				}
				return
			}
			if mode == "collision" {
				var failed *exec.ExitError
				if err != nil || !errors.As(result.runErr, &failed) || failed.ExitCode() != 1 || result.stdout.Len() != 0 || !strings.HasPrefix(result.stderr.String(), warning) || strings.Count(result.stderr.String(), "Warning:") != 1 || !strings.Contains(result.stderr.String(), "duplicate") {
					t.Fatal("collision must warn then fail", err, result.runErr, result.stdout.String(), result.stderr.String())
				}
				data, readErr := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
				if readErr != nil || string(data) != existing {
					t.Fatal("collision replaced destination", string(data), readErr)
				}
				if err := fixture.checkRequests(false); err != nil {
					t.Fatal(err)
				}
				return
			}
			if mode == "latent" {
				warning = `Warning: repository.settings.environments.deployment_branches_and_tags: observed {"protected_branches":true,"custom_branch_policies":true} is outside UI choices; using "no_restriction" (projection only). Environment "production".` + "\n"
			}
			if err != nil || result.runErr != nil || result.stdout.Len() != 0 || result.stderr.String() != warning {
				t.Fatal("export outcome and provenance", err, result.runErr, result.stdout.String(), result.stderr.String())
			}
			data, err := os.ReadFile(filepath.Join(fixture.work, ".github/settings.yml"))
			var document map[string]any
			if err != nil || yaml.Unmarshal(data, &document) != nil || strings.Contains(string(data), "future") || strings.Contains(string(data), "raw_value") || strings.Contains(string(data), "id:") {
				t.Fatal("raw observation leaked into YAML", string(data), err)
			}
			got := document["repository"].(map[string]any)["settings"].(map[string]any)["environments"]
			want := []any{map[string]any{
				"name":                           "production",
				"deployment_protection_rules":    map[string]any{"wait_timer": 0, "prevent_self_review": false, "allow_administrators_to_bypass_configured_protection_rules": false, "required_reviewers": []any{}},
				"deployment_branches_and_tags":   "selected_branches_and_tags",
				"deployment_branch_or_tag_rules": []any{map[string]any{"ref_type": "branch", "name_pattern": "main"}, map[string]any{"ref_type": "tag", "name_pattern": "main"}},
			}}
			if mode == "latent" {
				want[0].(map[string]any)["deployment_branches_and_tags"] = "no_restriction"
				delete(want[0].(map[string]any), "deployment_branch_or_tag_rules")
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatal("effective UI policy differs", got, want)
			}
			if err := fixture.checkRequests(false); err != nil {
				t.Fatal(err)
			}
		})
	}
}
