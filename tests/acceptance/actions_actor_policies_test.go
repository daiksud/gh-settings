//go:build acceptance

package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const actorPolicyInput = `version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      policies:
        - policy_name: policy-canary
          enforcement_status: disabled
          target_workflows:
            include: [ci.yml]
            exclude: []
          restrict_actors:
            allowed_actors:
              - user: 123
              - role: repository_admin
`

func actorPolicyPayload(t *testing.T, raw string) map[string]any {
	t.Helper()
	var p map[string]any
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestActionsActorsBuiltProductLifecycle(t *testing.T) {
	f := newActionsEventPoliciesFixture(t, "")
	f.write(t, actorPolicyInput)
	f.payloads = []map[string]any{
		actorPolicyPayload(t, `{"name":"policy-canary","enforcement":"disabled","conditions":{"workflow_path":{"include":["ci.yml"],"exclude":[]}},"rules":[{"type":"restrict_actions_actors","parameters":{"allowed_actors":[{"id":5,"type":"RepositoryRole"},{"id":123,"type":"User"}]}}]}`),
		actorPolicyPayload(t, `{"name":"policy-canary","enforcement":"disabled","rules":[{"type":"restrict_action_events","parameters":{"allowed_events":["push"]}},{"type":"restrict_actions_actors","parameters":{"allowed_actors":[{"id":5,"type":"RepositoryRole"},{"id":123,"type":"User"}]}}]}`),
		actorPolicyPayload(t, `{"name":"policy-canary","enforcement":"disabled","rules":[{"type":"restrict_action_events","parameters":{"allowed_events":["push"]}},{"type":"restrict_actions_actors","parameters":{"allowed_actors":[]}}]}`),
		actorPolicyPayload(t, `{"name":"policy-canary","enforcement":"disabled","rules":[{"type":"restrict_action_events","parameters":{"allowed_events":["push"]}}]}`),
	}
	f.run(t, false, "validate")
	out, _ := f.run(t, false, "plan")
	if !strings.Contains(out, "repository_admin") || !strings.Contains(out, `"user":123`) {
		t.Fatal(out)
	}
	report := filepath.Join(f.fixture.root, "actors-plan.json")
	f.run(t, false, "action", "plan", report)
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if json.Unmarshal(raw, &wire) != nil || wire["managed"] != float64(1) || len(wire["actions_policies"].([]any)) != 1 {
		t.Fatal(string(raw))
	}
	for _, mode := range []string{"detailed", "status-only"} {
		path := filepath.Join(f.fixture.root, mode+"-actors.md")
		args := []string{"action", "summary", report, path}
		if mode == "status-only" {
			args = append(args, mode)
		}
		f.run(t, false, args...)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "repository_admin") != (mode == "detailed") || mode == "status-only" && strings.Contains(string(data), "123") {
			t.Fatal(string(data))
		}
	}
	f.run(t, false, "apply")
	created := f.data(t)
	if !strings.Contains(created, "id: 17") {
		t.Fatal(created)
	}
	f.run(t, false, "apply")
	f.run(t, false, "export", "--actions-policies")
	exported := f.data(t)
	for _, want := range []string{"restrict_actors:", "repository_admin", "user: 123", "restrict_events:", "enabled: false", "ci.yml"} {
		if !strings.Contains(exported, want) {
			t.Fatal(exported)
		}
	}
	f.run(t, false, "validate")
	f.write(t, created)
	f.run(t, false, "apply")
	// Declared events must preserve omitted actors and targeting on an ID update.
	selected := `version: daiksud/gh-settings@v0
repository:
  settings:
    actions:
      policies:
        - id: 17
          policy_name: policy-canary
          enforcement_status: disabled
          restrict_events: {allowed_events: [push]}
`
	f.write(t, selected)
	f.run(t, false, "apply")
	f.write(t, selected+"          restrict_actors: {allowed_actors: []}\n")
	f.run(t, false, "apply")
	f.run(t, false, "export", "--actions-policies")
	if !strings.Contains(f.data(t), "allowed_actors: []") {
		t.Fatal(f.data(t))
	}
	f.write(t, selected+"          restrict_actors: {enabled: false}\n")
	f.run(t, false, "apply")
	f.run(t, false, "apply")
	f.mu.Lock()
	writes := f.writes
	conditions := f.objects[17]["conditions"]
	f.mu.Unlock()
	if writes != 4 || !reflect.DeepEqual(conditions, map[string]any{"workflow_path": map[string]any{"include": []any{"ci.yml"}, "exclude": []any{}}}) {
		t.Fatal(writes, conditions)
	}
	// An unsupported actor in an acquired rule prevents both export and selected update.
	preserved := f.data(t)
	f.mu.Lock()
	f.objects[17]["rules"] = []any{map[string]any{"type": "restrict_actions_actors", "parameters": map[string]any{"allowed_actors": []any{map[string]any{"type": "App", "id": 123}}}}}
	f.mu.Unlock()
	f.run(t, true, "export", "--actions-policies")
	if f.data(t) != preserved {
		t.Fatal("failed export replaced file")
	}
	f.run(t, true, "apply")
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writes != 4 {
		t.Fatal("wrote unsupported selected policy")
	}
}

func TestActionsActorsBuiltProductCreationRecovery(t *testing.T) {
	f := newActionsEventPoliciesFixture(t, "")
	f.write(t, actorPolicyInput)
	f.failure = "readback"
	f.payloads = []map[string]any{actorPolicyPayload(t, `{"name":"policy-canary","enforcement":"disabled","conditions":{"workflow_path":{"include":["ci.yml"],"exclude":[]}},"rules":[{"type":"restrict_actions_actors","parameters":{"allowed_actors":[{"id":5,"type":"RepositoryRole"},{"id":123,"type":"User"}]}}]}`)}
	_, diagnostic := f.run(t, true, "apply")
	if !strings.Contains(diagnostic, "id 17") || f.data(t) != actorPolicyInput {
		t.Fatal(diagnostic, f.data(t))
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writes != 1 {
		t.Fatal(f.writes)
	}
}
