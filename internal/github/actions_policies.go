package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type actionsPoliciesPreWriteFailure struct{ cause error }

func (e *actionsPoliciesPreWriteFailure) Error() string        { return e.cause.Error() }
func (e *actionsPoliciesPreWriteFailure) Unwrap() error        { return e.cause }
func (*actionsPoliciesPreWriteFailure) ReadFailedBeforeWrite() {}

// NewActionsPoliciesClient isolates the Actions policies API contract and refuses redirects.
func NewActionsPoliciesClient(host string) (*Client, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &actionsPoliciesPreWriteFailure{fmt.Errorf("configure Actions policies API: %w", err)}
	}
	return &Client{rest: rest}, nil
}
func actionsPoliciesPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/actions/policies"
}
func requiredActionsPolicy(fields map[string]json.RawMessage, key string, target any) error {
	raw, ok := fields[key]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("Actions policy missing/null %s", key)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("Actions policy %s: %w", key, err)
	}
	return nil
}
func actionsPolicyIdentity(raw map[string]json.RawMessage, repo repository.Repository) (int64, string, error) {
	var id int64
	var name, source, kind, target string
	for _, f := range []struct {
		k string
		v any
	}{{"id", &id}, {"name", &name}, {"source", &source}, {"source_type", &kind}, {"target", &target}} {
		if err := requiredActionsPolicy(raw, f.k, f.v); err != nil {
			return 0, "", err
		}
	}
	if id <= 0 || strings.TrimSpace(name) == "" {
		return 0, "", fmt.Errorf("invalid Actions policy identity")
	}
	if kind != "Repository" || !strings.EqualFold(source, repo.Owner+"/"+repo.Name) || target != "actions" {
		return 0, "", fmt.Errorf("Actions policy %d is not owned by target repository or has unexpected target", id)
	}
	return id, name, nil
}

// decodeActionsEventPolicy distinguishes complete but unrepresentable policies from
// incomplete responses. The former retain identity so unrelated policies stay untouched.
func decodeActionsEventPolicy(raw map[string]json.RawMessage, repo repository.Repository) (settings.ActionsEventPolicyState, error) {
	id, name, err := actionsPolicyIdentity(raw, repo)
	if err != nil {
		return settings.ActionsEventPolicyState{}, err
	}
	state := settings.ActionsEventPolicyState{ID: id, Name: name}
	unsupported := func(reason string) {
		if state.Unsupported == "" {
			state.Unsupported = reason
		}
	}
	for key := range raw {
		switch key {
		case "id", "name", "source", "source_type", "target", "enforcement", "conditions", "rules", "node_id", "_links", "created_at", "updated_at":
		default:
			unsupported("unknown Actions policy field " + key)
		}
	}
	var enforcement string
	if err := requiredActionsPolicy(raw, "enforcement", &enforcement); err != nil {
		return state, err
	}
	if enforcement != "active" && enforcement != "disabled" {
		unsupported("unsupported Actions policy enforcement " + enforcement)
	}
	var targeting *settings.ActionsWorkflowTarget
	if c, ok := raw["conditions"]; ok && !bytes.Equal(bytes.TrimSpace(c), []byte("null")) {
		var conditions map[string]json.RawMessage
		if err := json.Unmarshal(c, &conditions); err != nil || conditions == nil {
			return state, fmt.Errorf("invalid Actions policy conditions")
		}
		for key := range conditions {
			if key != "workflow_path" {
				unsupported("unsupported Actions policy condition " + key)
			}
		}
		if w, ok := conditions["workflow_path"]; ok {
			var workflow map[string]json.RawMessage
			if err := json.Unmarshal(w, &workflow); err != nil || workflow == nil {
				return state, fmt.Errorf("invalid Actions policy workflow_path")
			}
			for key := range workflow {
				if key != "include" && key != "exclude" {
					unsupported("unknown Actions policy workflow condition " + key)
				}
			}
			var include, exclude []string
			if err := requiredActionsPolicy(workflow, "include", &include); err != nil {
				return state, err
			}
			if err := requiredActionsPolicy(workflow, "exclude", &exclude); err != nil {
				return state, err
			}
			targeting = &settings.ActionsWorkflowTarget{Include: include, Exclude: exclude}
			if settings.EqualActionsWorkflowTargets(targeting, nil) {
				targeting = nil
			}
			if err := settings.ValidateActionsWorkflowTarget(targeting); err != nil {
				unsupported(err.Error())
			}
		}
	}
	var rules []map[string]json.RawMessage
	if err := requiredActionsPolicy(raw, "rules", &rules); err != nil {
		return state, err
	}
	if len(rules) == 0 || len(rules) > 2 {
		unsupported("Actions policy must contain one or two supported restriction rules")
	}
	var events []string
	var actors *[]settings.ActionsPolicyActor
	seenRules := map[string]bool{}
	for _, rule := range rules {
		var kind string
		if err := requiredActionsPolicy(rule, "type", &kind); err != nil {
			return state, err
		}
		if seenRules[kind] {
			unsupported("duplicate Actions policy rule " + kind)
		}
		seenRules[kind] = true
		if kind != "restrict_action_events" && kind != "restrict_actions_actors" {
			unsupported("unsupported Actions policy rule " + kind)
			continue
		}
		for key := range rule {
			if key != "type" && key != "parameters" {
				unsupported("unknown Actions policy rule field " + key)
			}
		}
		var params map[string]json.RawMessage
		if err := requiredActionsPolicy(rule, "parameters", &params); err != nil {
			return state, err
		}
		parameter := "allowed_events"
		if kind == "restrict_actions_actors" {
			parameter = "allowed_actors"
		}
		for key := range params {
			if key != parameter {
				unsupported("unknown Actions restriction parameter " + key)
			}
		}
		if kind == "restrict_action_events" {
			if err := requiredActionsPolicy(params, parameter, &events); err != nil {
				return state, err
			}
		} else {
			var rawActors []map[string]json.RawMessage
			if err := requiredActionsPolicy(params, parameter, &rawActors); err != nil {
				return state, err
			}
			values := make([]settings.ActionsPolicyActor, len(rawActors))
			for i, actor := range rawActors {
				if err := requiredActionsPolicy(actor, "id", &values[i].ID); err != nil {
					return state, err
				}
				if err := requiredActionsPolicy(actor, "type", &values[i].Type); err != nil {
					return state, err
				}
				for key := range actor {
					if key != "id" && key != "type" {
						unsupported("unknown Actions actor field " + key)
					}
				}
			}
			actors = &values
		}
	}
	policy := settings.ActionsEventPolicy{ID: id, Name: name, Enforcement: enforcement, Events: events, Actors: actors, TargetWorkflows: targeting}
	if state.Unsupported == "" {
		if err := settings.ValidateActionsEventPolicyState(policy); err != nil {
			unsupported(err.Error())
		} else {
			state.Policy = &policy
		}
	}
	return state, nil
}

func (client *Client) ReadActionsPolicies(repo repository.Repository) (settings.ActionsPolicyObservation, error) {
	fail := func(err error) (settings.ActionsPolicyObservation, error) {
		return settings.ActionsPolicyObservation{}, err
	}
	ids := []int64{}
	names := map[int64]string{}
	total := -1
	for page := 1; ; page++ {
		var raw map[string]json.RawMessage
		if err := client.rest.Get(fmt.Sprintf("%s?has_parents=false&per_page=100&page=%d", actionsPoliciesPath(repo), page), &raw); err != nil {
			return fail(fmt.Errorf("read Actions policies page: %w", err))
		}
		var count int
		var policies []map[string]json.RawMessage
		if err := requiredActionsPolicy(raw, "total_count", &count); err != nil {
			return fail(err)
		}
		if err := requiredActionsPolicy(raw, "policies", &policies); err != nil {
			return fail(err)
		}
		if count < 0 || (total >= 0 && count != total) || len(policies) > 100 {
			return fail(fmt.Errorf("unstable/invalid Actions policies total_count"))
		}
		total = count
		for _, item := range policies {
			id, name, err := actionsPolicyIdentity(item, repo)
			if err != nil {
				return fail(err)
			}
			if _, exists := names[id]; exists {
				return fail(fmt.Errorf("duplicate Actions policy ID %d", id))
			}
			ids = append(ids, id)
			names[id] = name
		}
		if len(ids) > total {
			return fail(fmt.Errorf("Actions policies exceeded total_count"))
		}
		if len(ids) == total {
			break
		}
		if len(policies) < 100 {
			return fail(fmt.Errorf("incomplete Actions policies collection"))
		}
	}
	values := make([]settings.ActionsEventPolicyState, 0, len(ids))
	for _, id := range ids {
		state, err := client.readActionsEventPolicy(repo, id)
		if err != nil {
			return fail(err)
		}
		if state.Name != names[id] {
			return fail(fmt.Errorf("Actions policy name changed during read"))
		}
		values = append(values, state)
	}
	return settings.NewActionsPolicyObservation(values)
}
func (client *Client) readActionsEventPolicy(repo repository.Repository, id int64) (settings.ActionsEventPolicyState, error) {
	var raw map[string]json.RawMessage
	if err := client.rest.Get(fmt.Sprintf("%s/%d", actionsPoliciesPath(repo), id), &raw); err != nil {
		return settings.ActionsEventPolicyState{}, fmt.Errorf("read Actions policy %d: %w", id, err)
	}
	state, err := decodeActionsEventPolicy(raw, repo)
	if err != nil {
		return state, err
	}
	if state.ID != id {
		return state, fmt.Errorf("Actions policy ID changed during read")
	}
	return state, nil
}
func equalActionsEventPolicy(a, b settings.ActionsEventPolicy) bool {
	a.Events = slices.Clone(a.Events)
	b.Events = slices.Clone(b.Events)
	slices.Sort(a.Events)
	slices.Sort(b.Events)
	return a.ID == b.ID && a.Name == b.Name && a.Enforcement == b.Enforcement && slices.Equal(a.Events, b.Events) && equalActionsPolicyActors(a.Actors, b.Actors) && settings.EqualActionsWorkflowTargets(a.TargetWorkflows, b.TargetWorkflows)
}
func equalActionsPolicyActors(a, b *[]settings.ActionsPolicyActor) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	left, right := slices.Clone(*a), slices.Clone(*b)
	compare := func(a, b settings.ActionsPolicyActor) int {
		if order := strings.Compare(a.Type, b.Type); order != 0 {
			return order
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	}
	slices.SortFunc(left, compare)
	slices.SortFunc(right, compare)
	return slices.Equal(left, right)
}
func (client *Client) WriteActionsEventPolicy(repo repository.Repository, change settings.ActionsEventPolicyChange) (int64, error) {
	desired := change.After()
	if err := settings.ValidateActionsEventPolicyState(desired); err != nil {
		return 0, &actionsPoliciesPreWriteFailure{err}
	}
	op := change.Operation()
	if op != "create" && op != "update" {
		return 0, fmt.Errorf("unsupported Actions policy operation %q", op)
	}
	if op == "update" {
		if change.ID() <= 0 || desired.ID != change.ID() {
			return 0, fmt.Errorf("invalid Actions policy update identity")
		}
		current, err := client.readActionsEventPolicy(repo, change.ID())
		if err != nil {
			return 0, &actionsPoliciesPreWriteFailure{err}
		}
		before := change.Before()
		if current.Policy == nil || before == nil || !equalActionsEventPolicy(*current.Policy, *before) {
			return 0, &actionsPoliciesPreWriteFailure{fmt.Errorf("Actions policy changed or is unsupported; refusing update")}
		}
	} else if desired.ID != 0 {
		return 0, fmt.Errorf("Actions policy create cannot contain ID")
	}
	rules := []any{}
	if desired.Events != nil {
		rules = append(rules, map[string]any{"type": "restrict_action_events", "parameters": map[string]any{"allowed_events": desired.Events}})
	}
	if desired.Actors != nil {
		actors := make([]map[string]any, len(*desired.Actors))
		for i, actor := range *desired.Actors {
			actors[i] = map[string]any{"id": actor.ID, "type": actor.Type}
		}
		rules = append(rules, map[string]any{"type": "restrict_actions_actors", "parameters": map[string]any{"allowed_actors": actors}})
	}
	payload := map[string]any{"name": desired.Name, "enforcement": desired.Enforcement, "rules": rules}
	// Omit unchanged conditions: GitHub preserves targeting on updates. Explicit
	// changes use both required arrays; ~ALL is the nonempty All workflows reset.
	if desired.TargetWorkflows != nil && (change.Before() == nil || !settings.EqualActionsWorkflowTargets(desired.TargetWorkflows, change.Before().TargetWorkflows)) {
		target := settings.CanonicalActionsWorkflowTarget(desired.TargetWorkflows)
		payload["conditions"] = map[string]any{"workflow_path": map[string]any{"include": target.Include, "exclude": target.Exclude}}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, &actionsPoliciesPreWriteFailure{err}
	}
	path := actionsPoliciesPath(repo)
	method := "POST"
	if op == "update" {
		path += fmt.Sprintf("/%d", change.ID())
		method = "PUT"
	}
	var response map[string]json.RawMessage
	if err := client.rest.Do(method, path, bytes.NewReader(data), &response); err != nil {
		return 0, fmt.Errorf("write Actions policy (outcome may be uncertain; do not retry creation): %w", err)
	}
	id, name, err := actionsPolicyIdentity(response, repo)
	if err != nil {
		return 0, fmt.Errorf("invalid Actions policy write acknowledgment: %w", err)
	}
	if op == "update" && id != change.ID() {
		return 0, fmt.Errorf("Actions policy acknowledgment ID mismatch")
	}
	// A confirmed created identity survives acknowledgment/readback failures for recovery.
	if name != desired.Name {
		return id, fmt.Errorf("Actions policy acknowledgment name mismatch")
	}
	ack, err := decodeActionsEventPolicy(response, repo)
	if err != nil {
		return id, fmt.Errorf("invalid Actions policy acknowledgment: %w", err)
	}
	desired.ID = id
	if ack.Policy == nil || !equalActionsEventPolicy(*ack.Policy, desired) {
		return id, fmt.Errorf("Actions policy acknowledgment differs from requested policy")
	}
	actual, err := client.readActionsEventPolicy(repo, id)
	if err != nil {
		return id, fmt.Errorf("verify Actions policy after write: %w", err)
	}
	if actual.Policy == nil || !equalActionsEventPolicy(*actual.Policy, desired) {
		return id, fmt.Errorf("Actions policy readback differs from requested policy")
	}
	return id, nil
}
