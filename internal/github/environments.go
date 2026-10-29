package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type EnvironmentsClient struct {
	rest                 *api.RESTClient
	NormalizationWarning func(string, settings.EnvironmentDeploymentNormalization)
}

type environmentPreWriteFailure struct{ cause error }

func (e *environmentPreWriteFailure) Error() string        { return e.cause.Error() }
func (e *environmentPreWriteFailure) Unwrap() error        { return e.cause }
func (*environmentPreWriteFailure) ReadFailedBeforeWrite() {}

func NewEnvironmentsClient(host string) (*EnvironmentsClient, error) {
	c, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, fmt.Errorf("configure Environments API: %w", err)
	}
	return &EnvironmentsClient{rest: c}, nil
}

func environmentsPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/environments"
}
func environmentPath(repo repository.Repository, name string) string {
	return environmentsPath(repo) + "/" + url.PathEscape(name)
}

func requiredEnvironmentJSON(raw map[string]json.RawMessage, key string, target any) error {
	v, ok := raw[key]
	if !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
		return fmt.Errorf("environment response missing/null %s", key)
	}
	if err := json.Unmarshal(v, target); err != nil {
		return fmt.Errorf("invalid environment %s: %w", key, err)
	}
	return nil
}

func environmentKnownFields(raw map[string]json.RawMessage, allowed ...string) error {
	for key := range raw {
		known := false
		for _, candidate := range allowed {
			if key == candidate {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("unsupported environment response field %q; refusing to replace unknown settings", key)
		}
	}
	return nil
}

// A present, complete protection_rules array describes the enabled built-in
// rules. Absence of a rule in that array means disabled, not a failed read.
func decodeEnvironment(raw map[string]json.RawMessage) (settings.EnvironmentState, error) {
	var result settings.EnvironmentState
	if err := environmentKnownFields(raw, "id", "node_id", "name", "url", "html_url", "created_at", "updated_at", "protection_rules", "deployment_branch_policy", "can_admins_bypass"); err != nil {
		return result, err
	}
	if err := requiredEnvironmentJSON(raw, "id", &result.ID); err != nil {
		return result, err
	}
	if err := requiredEnvironmentJSON(raw, "name", &result.Policy.Name); err != nil {
		return result, err
	}
	var bypass bool
	if err := requiredEnvironmentJSON(raw, "can_admins_bypass", &bypass); err != nil {
		return result, err
	}
	result.Policy.CanAdminsBypass = &bypass
	var rules []map[string]json.RawMessage
	if err := requiredEnvironmentJSON(raw, "protection_rules", &rules); err != nil {
		return result, err
	}
	timer, prevent, reviewers := 0, false, []settings.EnvironmentReviewer{}
	seen := map[string]bool{}
	for _, rule := range rules {
		var kind string
		if err := requiredEnvironmentJSON(rule, "type", &kind); err != nil {
			return result, err
		}
		if seen[kind] {
			return result, fmt.Errorf("duplicate environment protection rule %q", kind)
		}
		seen[kind] = true
		switch kind {
		case "wait_timer":
			if err := environmentKnownFields(rule, "id", "node_id", "type", "wait_timer"); err != nil {
				return result, err
			}
			if err := requiredEnvironmentJSON(rule, "wait_timer", &timer); err != nil {
				return result, err
			}
		case "required_reviewers":
			if err := environmentKnownFields(rule, "id", "node_id", "type", "prevent_self_review", "reviewers"); err != nil {
				return result, err
			}
			if err := requiredEnvironmentJSON(rule, "prevent_self_review", &prevent); err != nil {
				return result, err
			}
			var entries []map[string]json.RawMessage
			if err := requiredEnvironmentJSON(rule, "reviewers", &entries); err != nil {
				return result, err
			}
			for _, entry := range entries {
				if err := environmentKnownFields(entry, "type", "reviewer"); err != nil {
					return result, err
				}
				var reviewer settings.EnvironmentReviewer
				var principal map[string]json.RawMessage
				if err := requiredEnvironmentJSON(entry, "type", &reviewer.Type); err != nil {
					return result, err
				}
				if err := requiredEnvironmentJSON(entry, "reviewer", &principal); err != nil {
					return result, err
				}
				if err := requiredEnvironmentJSON(principal, "id", &reviewer.ID); err != nil {
					return result, err
				}
				if _, exists := principal["type"]; exists {
					var principalType string
					if err := requiredEnvironmentJSON(principal, "type", &principalType); err != nil {
						return result, err
					}
					if principalType != reviewer.Type {
						return result, fmt.Errorf("environment reviewer type differs from principal")
					}
				}
				// Principal profile fields are read-only metadata, not environment
				// protection settings. Only the typed numeric reference is writable.
				reviewers = append(reviewers, reviewer)
			}
		case "branch_policy":
			if err := environmentKnownFields(rule, "id", "node_id", "type"); err != nil {
				return result, err
			}
		default:
			return result, fmt.Errorf("unsupported environment protection rule %q", kind)
		}
	}
	result.Policy.WaitTimer, result.Policy.PreventSelfReview, result.Policy.Reviewers = &timer, &prevent, &reviewers
	branch, exists := raw["deployment_branch_policy"]
	if !exists {
		return result, fmt.Errorf("environment response missing deployment_branch_policy")
	}
	result.Policy.DeploymentBranchPolicyDeclared = true
	if !bytes.Equal(bytes.TrimSpace(branch), []byte("null")) {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(branch, &fields); err != nil {
			return result, err
		}
		if err := environmentKnownFields(fields, "protected_branches", "custom_branch_policies"); err != nil {
			return result, err
		}
		policy := &settings.EnvironmentBranchPolicy{}
		if err := requiredEnvironmentJSON(fields, "protected_branches", &policy.ProtectedBranches); err != nil {
			return result, err
		}
		if err := requiredEnvironmentJSON(fields, "custom_branch_policies", &policy.CustomBranchPolicies); err != nil {
			return result, err
		}
		result.Policy.DeploymentBranchPolicy = policy
	}
	// Validate complete observations without declaration defaulting.
	observation, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{result})
	if err != nil {
		return result, err
	}
	values, err := observation.Values()
	if err != nil {
		return result, err
	}
	return values[0], nil
}

func (c *EnvironmentsClient) readOne(repo repository.Repository, name string) (settings.EnvironmentState, error) {
	var raw map[string]json.RawMessage
	if err := c.rest.Get(environmentPath(repo, name), &raw); err != nil {
		return settings.EnvironmentState{}, fmt.Errorf("read environment %q (unavailable is not absent): %w", name, err)
	}
	state, err := decodeEnvironment(raw)
	if err != nil {
		return state, fmt.Errorf("read environment %q: %w", name, err)
	}
	if settings.EnvironmentIdentity(state.Policy.Name) != settings.EnvironmentIdentity(name) {
		return state, fmt.Errorf("environment response name differs from selected name")
	}
	var custom map[string]json.RawMessage
	if err := c.rest.Get(environmentPath(repo, name)+"/deployment_protection_rules", &custom); err != nil {
		return state, fmt.Errorf("read custom environment protections: %w", err)
	}
	var count int
	var rules []json.RawMessage
	if err := requiredEnvironmentJSON(custom, "total_count", &count); err != nil {
		return state, err
	}
	if err := requiredEnvironmentJSON(custom, "custom_deployment_protection_rules", &rules); err != nil {
		return state, err
	}
	if count != 0 || len(rules) != 0 {
		return state, fmt.Errorf("custom environment protection rules are not supported; no environment write is safe")
	}
	if policy := state.Policy.DeploymentBranchPolicy; policy != nil && policy.CustomBranchPolicies {
		patterns, err := c.readPatterns(repo, name)
		if err != nil {
			return state, err
		}
		state.Patterns = patterns
		values := make([]settings.EnvironmentPattern, 0, len(patterns))
		for _, pattern := range patterns {
			values = append(values, pattern.Pattern)
		}
		state.Policy.DeploymentBranchPatterns = &values
	}
	return state, nil
}

// Read inventories names completely before selecting detail reads. A nil names
// slice requests all environments for export; an empty slice selects nothing.
// A failed inventory/detail/custom-protection read never becomes absence.
func (c *EnvironmentsClient) Read(repo repository.Repository, names []string) (settings.EnvironmentObservation, error) {
	if names != nil && len(names) == 0 {
		return settings.NewEnvironmentObservation(nil)
	}
	selected := map[string]bool{}
	for _, name := range names {
		selected[settings.EnvironmentIdentity(name)] = true
	}
	seenNames, seenIDs := map[string]bool{}, map[int64]bool{}
	states := []settings.EnvironmentState{}
	expectedTotal, observed := -1, 0
	for page := 1; ; page++ {
		var raw map[string]json.RawMessage
		if err := c.rest.Get(fmt.Sprintf("%s?per_page=100&page=%d", environmentsPath(repo), page), &raw); err != nil {
			return settings.EnvironmentObservation{}, fmt.Errorf("list environments: %w", err)
		}
		var count int
		var entries []map[string]json.RawMessage
		if err := requiredEnvironmentJSON(raw, "total_count", &count); err != nil {
			return settings.EnvironmentObservation{}, err
		}
		if err := requiredEnvironmentJSON(raw, "environments", &entries); err != nil {
			return settings.EnvironmentObservation{}, err
		}
		if count < 0 || (expectedTotal >= 0 && expectedTotal != count) {
			return settings.EnvironmentObservation{}, fmt.Errorf("environment inventory count changed or is invalid")
		}
		expectedTotal = count
		for _, entry := range entries {
			var name string
			var id int64
			if err := requiredEnvironmentJSON(entry, "name", &name); err != nil {
				return settings.EnvironmentObservation{}, err
			}
			if err := requiredEnvironmentJSON(entry, "id", &id); err != nil {
				return settings.EnvironmentObservation{}, err
			}
			key := settings.EnvironmentIdentity(name)
			if name == "" || id <= 0 || seenNames[key] || seenIDs[id] {
				return settings.EnvironmentObservation{}, fmt.Errorf("ambiguous environment inventory")
			}
			seenNames[key], seenIDs[id] = true, true
			observed++
			if names == nil || selected[key] {
				state, err := c.readOne(repo, name)
				if err != nil {
					return settings.EnvironmentObservation{}, err
				}
				states = append(states, state)
			}
		}
		if observed > count {
			return settings.EnvironmentObservation{}, fmt.Errorf("environment inventory exceeds declared count")
		}
		if observed == count {
			break
		}
		if len(entries) != 100 {
			return settings.EnvironmentObservation{}, fmt.Errorf("incomplete environment inventory")
		}
	}
	return settings.NewEnvironmentObservation(states)
}

// Update uses ordinary name-addressed upsert. It neither binds the environment
// to an old numeric ID nor claims atomic protection against concurrent changes.
func (c *EnvironmentsClient) Update(repo repository.Repository, desired settings.EnvironmentDeclaration) (bool, error) {
	pre := func(err error) (bool, error) { return false, &environmentPreWriteFailure{err} }
	d, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{desired})
	if err != nil {
		return pre(err)
	}
	identity := &Client{rest: c.rest}
	initial, err := identity.ReadArchive(repo)
	if err != nil {
		return pre(err)
	}
	if initial.Archived {
		return pre(fmt.Errorf("archived repository cannot update environments without explicit unarchive"))
	}
	current, err := c.Read(repo, []string{desired.Name})
	if err != nil {
		return pre(err)
	}
	if desired.DeploymentBranchPatterns != nil && desired.DeploymentBranchPolicy == nil {
		states, _ := current.Values()
		if len(states) != 1 || states[0].Policy.DeploymentBranchPolicy == nil || !states[0].Policy.DeploymentBranchPolicy.CustomBranchPolicies {
			return pre(fmt.Errorf("deployment patterns without an explicit mode require an existing custom-mode environment"))
		}
	}
	changes, err := settings.ReconcileEnvironments(current, &d)
	if err != nil {
		return pre(err)
	}
	if len(changes) == 0 {
		return false, nil
	}
	expected := changes[0].After()
	body, err := json.Marshal(expected)
	if err != nil {
		return pre(err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return pre(err)
	}
	delete(payload, "name")
	delete(payload, "deployment_branch_patterns")
	body, err = json.Marshal(payload)
	if err != nil {
		return pre(err)
	}
	fresh, err := identity.ReadArchive(repo)
	if err != nil {
		return pre(err)
	}
	if fresh != initial {
		return pre(fmt.Errorf("repository identity or archive state changed before environment update"))
	}
	states, err := current.Values()
	if err != nil {
		return pre(err)
	}
	for _, state := range states {
		c.warnDeploymentNormalization(state, *expected)
	}
	var raw map[string]json.RawMessage
	if err := c.rest.Put(environmentPath(repo, expected.Name), bytes.NewReader(body), &raw); err != nil {
		return false, fmt.Errorf("update environment %q (write may have applied): %w", expected.Name, err)
	}
	if err := verifyEnvironmentUpdateResponse(raw, *expected); err != nil {
		return false, fmt.Errorf("verify environment update response (write may have applied): %w", err)
	}
	if expected.DeploymentBranchPatterns != nil {
		// Mode entry and pattern changes are separate API calls. No transaction,
		// intermediate enforcement guarantee, or automatic rollback is claimed.
		state, err := c.readOne(repo, expected.Name)
		if err != nil {
			return false, fmt.Errorf("read patterns after environment PUT (write may have applied): %w", err)
		}
		c.warnDeploymentNormalization(state, *expected)
		if state.Policy.DeploymentBranchPolicy == nil || !state.Policy.DeploymentBranchPolicy.CustomBranchPolicies {
			return false, fmt.Errorf("environment is not in custom mode after PUT (write may have applied)")
		}
		if err := c.applyPatterns(repo, expected.Name, state.Patterns, expected.DeploymentBranchPatterns); err != nil {
			return false, fmt.Errorf("update deployment patterns (earlier writes remain applied): %w", err)
		}
	}
	actual, err := c.readOne(repo, expected.Name)
	if err != nil {
		return false, fmt.Errorf("verify environment readback (write may have applied): %w", err)
	}
	c.warnDeploymentNormalization(actual, *expected)
	if err := verifyEnvironmentRequestedState(actual, *expected); err != nil {
		return false, fmt.Errorf("verify environment readback (write may have applied): %w", err)
	}
	return true, nil
}
