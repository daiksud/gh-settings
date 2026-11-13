//go:build acceptance

package acceptance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const actionsEventPoliciesEndpoint = "/repos/owner/current/actions/policies"
const actionsEventPoliciesInput = `# consumer intent
version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      policies:
        - policy_name: policy-canary # keep comment
          enforcement_status: active
          restrict_events: {allowed_events: [push]}
`

type actionsEventPoliciesFixture struct {
	mu       sync.Mutex
	fixture  *apiFixture
	env      []string
	objects  map[int]map[string]any
	calls    []string
	writes   int
	failure  string
	payloads []map[string]any
}

func newActionsEventPoliciesFixture(t *testing.T, mode string) *actionsEventPoliciesFixture {
	t.Helper()
	f := &actionsEventPoliciesFixture{objects: map[int]map[string]any{}}
	fixture, err := startHTTPFixture(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, actionsEventPoliciesEndpoint) {
			if r.Header.Get("X-GitHub-Api-Version") != "2026-03-10" {
				t.Error("wrong API version")
			}
			if r.URL.Path == actionsEventPoliciesEndpoint && r.Method == "GET" {
				if r.URL.Query().Get("has_parents") != "false" || r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("page") != "1" {
					t.Error("wrong collection query", r.URL)
				}
				items := []any{}
				for _, id := range []int{17, 99} {
					if object := f.objects[id]; object != nil {
						items = append(items, object)
					}
				}
				count := len(items)
				if f.failure == "incomplete" {
					count++
				}
				json.NewEncoder(w).Encode(map[string]any{"total_count": count, "policies": items})
				return
			}
			id := 17
			if r.URL.Path == actionsEventPoliciesEndpoint+"/99" {
				id = 99
			}
			if r.Method == "GET" && f.objects[id] != nil {
				if f.failure == "readback" && f.writes > 0 {
					w.WriteHeader(503)
					fmt.Fprint(w, `{}`)
					return
				}
				if f.failure == "target-readback" && f.writes > 0 {
					f.objects[id]["conditions"] = map[string]any{"workflow_path": map[string]any{"include": []any{"wrong.yml"}, "exclude": []any{}}}
				}
				json.NewEncoder(w).Encode(f.objects[id])
				return
			}
			if (r.Method == "POST" && r.URL.Path == actionsEventPoliciesEndpoint) || (r.Method == "PUT" && r.URL.Path == actionsEventPoliciesEndpoint+"/17") {
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				want := map[string]any{"name": "policy-canary", "enforcement": payload["enforcement"], "rules": []any{map[string]any{"type": "restrict_action_events", "parameters": map[string]any{"allowed_events": []any{"push"}}}}}
				if len(f.payloads) > 0 {
					if f.writes >= len(f.payloads) {
						t.Errorf("unexpected extra policy write: %v", payload)
						w.WriteHeader(500)
						return
					}
					want = f.payloads[f.writes]
				}
				if !reflect.DeepEqual(payload, want) || payload["enforcement"] != "active" && payload["enforcement"] != "disabled" {
					t.Error("unexpected payload", payload)
				}
				if r.Method == "PUT" && payload["conditions"] == nil && f.objects[17] != nil {
					if conditions := f.objects[17]["conditions"]; conditions != nil {
						payload["conditions"] = conditions
					}
				}
				f.writes++
				payload["id"], payload["source_type"], payload["source"], payload["target"] = 17, "Repository", "owner/current", "actions"
				f.objects[17] = payload
				if f.failure == "writeback" {
					path := filepath.Join(f.fixture.work, ".github/settings.yml")
					data, err := os.ReadFile(path)
					if err != nil {
						t.Error(err)
					}
					if err := os.WriteFile(path, append(data, []byte("# concurrent edit\n")...), 0600); err != nil {
						t.Error(err)
					}
				}
				if r.Method == "POST" {
					w.WriteHeader(201)
				}
				json.NewEncoder(w).Encode(payload)
				return
			}
			t.Errorf("unexpected policy request %s %s", r.Method, r.URL)
			w.WriteHeader(400)
			return
		}
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
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(400)
		}
	}, false, "/actions/policies", "/labels", "/topics")
	if err != nil {
		t.Fatal(err)
	}
	f.fixture = fixture
	t.Cleanup(fixture.close)
	f.env = append(labelIDFixtureEnv(fixture.cfg, mode == "actions"), "SETTINGS_TARGET=owner/current", "SETTINGS_ACTOR=user", "SETTINGS_TRIGGERING_ACTOR=user", "SETTINGS_EVENT=push", "SETTINGS_RUN_ID=123", "SETTINGS_RUN_ATTEMPT=1", "SETTINGS_WORKFLOW_REF=owner/current/.github/workflows/settings.yml@refs/heads/main", "SETTINGS_WORKFLOW_SHA="+strings.Repeat("a", 40), "SETTINGS_CONFIG_SHA="+strings.Repeat("b", 40), "SETTINGS_CLI_VERSION=dev", "SETTINGS_CLI_SHA256="+strings.Repeat("c", 64))
	if err := setupOrdinaryOrigin(fixture.work, f.env); err != nil {
		t.Fatal(err)
	}
	f.write(t, actionsEventPoliciesInput)
	return f
}
func (f *actionsEventPoliciesFixture) write(t *testing.T, input string) {
	t.Helper()
	if err := writeSettingsFile(f.fixture.work, input); err != nil {
		t.Fatal(err)
	}
}
func (f *actionsEventPoliciesFixture) run(t *testing.T, fail bool, args ...string) (string, string) {
	t.Helper()
	r, err := executeCLI(acceptanceBinary(t), f.fixture.work, f.fixture.root, f.env, args, false)
	if err != nil || (r.runErr != nil) != fail || !fail && hasNonInfoDiagnostic(r.stderr.String()) || fail && r.stdout.Len() != 0 {
		t.Fatalf("args=%v exit=%v stdout=%q stderr=%q error=%v", args, r.runErr, r.stdout.String(), r.stderr.String(), err)
	}
	return r.stdout.String(), withoutExportInfo(r.stderr.String())
}
func (f *actionsEventPoliciesFixture) data(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.fixture.work, ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func (f *actionsEventPoliciesFixture) expectPolicyCalls(t *testing.T, want ...string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	var calls []string
	for _, call := range f.calls {
		if strings.Contains(call, actionsEventPoliciesEndpoint) {
			calls = append(calls, call)
		}
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("policy calls=%v want=%v", calls, want)
	}
	f.calls = nil
}

func TestActionsEventPoliciesValidatePlanAndSummaryProcess(t *testing.T) {
	f := newActionsEventPoliciesFixture(t, "")
	f.run(t, false, "validate")
	f.mu.Lock()
	if len(f.calls) != 0 {
		t.Error("validate contacted API", f.calls)
	}
	f.mu.Unlock()
	out, _ := f.run(t, false, "plan")
	if !strings.Contains(out, "policy-canary") || !strings.Contains(out, "repository.settings.actions.policies") {
		t.Fatal(out)
	}
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint)
	report := filepath.Join(f.fixture.root, "plan.json")
	f.run(t, false, "action", "plan", report)
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint)
	raw, err := os.ReadFile(report)
	var wire map[string]any
	if err != nil || json.Unmarshal(raw, &wire) != nil || wire["managed"] != float64(1) || wire["actions_policies"] == nil || wire["outcome"] != "changed" {
		t.Fatal(string(raw), err)
	}
	if info, err := os.Stat(report); err != nil || info.Mode().Perm() != 0600 {
		t.Fatal(info, err)
	}
	for _, mode := range []string{"detailed", "status-only"} {
		summary := filepath.Join(f.fixture.root, mode+".md")
		args := []string{"action", "summary", report, summary}
		if mode == "status-only" {
			args = append(args, mode)
		}
		out, _ = f.run(t, false, args...)
		wantOutput := ""
		if mode == "status-only" {
			wantOutput = "plan-outcome=changed\n"
		}
		if out != wantOutput {
			t.Fatalf("summary output=%q want=%q", out, wantOutput)
		}
		raw, err = os.ReadFile(summary)
		if err != nil || strings.Contains(string(raw), "policy-canary") != (mode == "detailed") || mode == "status-only" && !strings.Contains(string(raw), "changed") {
			t.Fatal(string(raw), err)
		}
	}
	f.expectPolicyCalls(t)
	f.run(t, false, "action", "plan", report, "status-only")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint)
	raw, err = os.ReadFile(report)
	if err != nil || json.Unmarshal(raw, &wire) != nil || wire["managed"] != float64(1) || wire["actions_policies"] == nil {
		t.Fatal(string(raw), err)
	}
	if f.data(t) != actionsEventPoliciesInput {
		t.Fatal("plan modified configuration")
	}
}

func TestActionsEventPoliciesCreateRecoveryUpdateProcess(t *testing.T) {
	for _, mode := range []string{"writeback", "no-writeback", "actions"} {
		t.Run(mode, func(t *testing.T) {
			f := newActionsEventPoliciesFixture(t, mode)
			// An unsupported, unlisted policy must retain its complete identity and contents.
			unlisted := map[string]any{"id": 99, "name": "untouched", "source_type": "Repository", "source": "owner/current", "target": "actions", "enforcement": "active", "conditions": map[string]any{"workflow_path": map[string]any{"include": []any{"selected.yml"}, "exclude": []any{}}}, "rules": []any{}}
			f.objects[99] = unlisted
			args := []string{"apply"}
			if mode == "no-writeback" {
				args = append(args, "--no-writeback")
			}
			out, _ := f.run(t, false, args...)
			if !strings.Contains(out, `Created repository.settings.actions.policies "policy-canary": id 17`) {
				t.Fatal(out)
			}
			f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/99", "POST "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17")
			if mode == "writeback" {
				data := f.data(t)
				for _, want := range []string{"id: 17", "# consumer intent", "# keep comment"} {
					if !strings.Contains(data, want) {
						t.Fatal(data)
					}
				}
				f.run(t, false, args...)
				f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17", "GET "+actionsEventPoliciesEndpoint+"/99")
				for _, enforcement := range []string{"disabled", "active"} {
					data = strings.Replace(data, "enforcement_status: active", "enforcement_status: disabled", 1)
					if enforcement == "active" {
						data = strings.Replace(data, "enforcement_status: disabled", "enforcement_status: active", 1)
					}
					f.write(t, data)
					f.run(t, false, "apply")
					f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17", "GET "+actionsEventPoliciesEndpoint+"/99", "GET "+actionsEventPoliciesEndpoint+"/17", "PUT "+actionsEventPoliciesEndpoint+"/17", "GET "+actionsEventPoliciesEndpoint+"/17")
				}
			} else {
				if f.data(t) != actionsEventPoliciesInput {
					t.Fatal("disabled writeback modified file")
				}
				_, diagnostic := f.run(t, true, args...)
				if !strings.Contains(diagnostic, "already belongs to id 17") || !strings.Contains(diagnostic, "specify its id") {
					t.Fatal(diagnostic)
				}
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			wantWrites := 1
			if mode == "writeback" {
				wantWrites = 3
			}
			if f.writes != wantWrites || !reflect.DeepEqual(f.objects[99], unlisted) {
				t.Fatal("unexpected mutation", f.writes, f.objects)
			}
		})
	}
}

func TestActionsEventPoliciesPartialCreateRecoveryProcess(t *testing.T) {
	for _, failure := range []string{"writeback", "readback"} {
		t.Run(failure, func(t *testing.T) {
			f := newActionsEventPoliciesFixture(t, "")
			f.failure = failure
			_, diagnostic := f.run(t, true, "apply")
			if !strings.Contains(diagnostic, "id 17") {
				t.Fatal("lost recoverable identity", diagnostic)
			}
			if failure == "writeback" {
				if !strings.Contains(diagnostic, "local ID writeback failed") || f.data(t) != actionsEventPoliciesInput+"# concurrent edit\n" {
					t.Fatal(diagnostic, f.data(t))
				}
			} else if f.data(t) != actionsEventPoliciesInput || !strings.Contains(diagnostic, "verify Actions policy") {
				t.Fatal(diagnostic, f.data(t))
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.writes != 1 {
				t.Fatal(f.writes)
			}
		})
	}
}

func TestActionsEventPoliciesExportIsolationAndSafetyProcess(t *testing.T) {
	f := newActionsEventPoliciesFixture(t, "")
	f.run(t, false, "apply")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "POST "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17")
	f.run(t, false, "export")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17")
	data := f.data(t)
	for _, want := range []string{"id: 17", "policy_name: policy-canary", "enforcement_status: active", "restrict_events:"} {
		if !strings.Contains(data, want) {
			t.Fatal(data)
		}
	}
	if strings.Contains(data, "allowed_actions:") {
		t.Fatal("export coupled independent policy groups", data)
	}
	f.run(t, false, "validate")
	f.expectPolicyCalls(t)
	for _, failure := range []string{"unsupported", "incomplete"} {
		f.mu.Lock()
		f.failure = failure
		if failure == "unsupported" {
			f.objects[17]["conditions"] = map[string]any{"unsupported_condition": map[string]any{}}
		}
		f.mu.Unlock()
		f.run(t, true, "export")
		if f.data(t) != data {
			t.Fatal("failed export overwrote destination")
		}
		if failure == "unsupported" {
			_, diagnostic := f.run(t, true, "plan")
			if !strings.Contains(diagnostic, "unsupported") && !strings.Contains(diagnostic, "selected workflows") {
				t.Fatal(diagnostic)
			}
		}
	}
}
