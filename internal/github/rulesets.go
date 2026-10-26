package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// NewRulesetsClient uses normal gh credentials, bounded requests and no redirects.
// Pin this resource's contract independently of unrelated existing API adapters.
func NewRulesetsClient(host string) (*Client, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, fmt.Errorf("configure Rulesets API: %w", err)
	}
	return &Client{rest: rest}, nil
}
func rulesetsPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/rulesets"
}
func requiredRuleJSON(fields map[string]json.RawMessage, key string, target any) error {
	raw, ok := fields[key]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("Rulesets response missing/null %s", key)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("Rulesets response %s: %w", key, err)
	}
	return nil
}
func rulesetOwner(raw map[string]json.RawMessage, repo repository.Repository) (bool, error) {
	var owner, source string
	if err := requiredRuleJSON(raw, "source_type", &owner); err != nil {
		return false, err
	}
	if err := requiredRuleJSON(raw, "source", &source); err != nil {
		return false, err
	}
	switch owner {
	case "Organization", "Enterprise":
		if source == "" {
			return false, fmt.Errorf("missing inherited Ruleset source")
		}
		return false, nil
	case "Repository":
		if !strings.EqualFold(source, repo.Owner+"/"+repo.Name) {
			return false, fmt.Errorf("Ruleset belongs to another repository")
		}
		return true, nil
	default:
		return false, fmt.Errorf("unknown Ruleset source_type %q", owner)
	}
}
func (client *Client) ReadRulesets(repo repository.Repository) (settings.RulesetObservation, error) {
	ids := []int64{}
	seen := map[int64]bool{}
	for page := 1; ; page++ {
		var items []map[string]json.RawMessage
		if err := client.rest.Get(fmt.Sprintf("%s?includes_parents=false&per_page=100&page=%d", rulesetsPath(repo), page), &items); err != nil {
			return settings.RulesetObservation{}, fmt.Errorf("read Rulesets page: %w", err)
		}
		if items == nil {
			return settings.RulesetObservation{}, fmt.Errorf("null Rulesets list")
		}
		for _, item := range items {
			owned, err := rulesetOwner(item, repo)
			if err != nil {
				return settings.RulesetObservation{}, err
			}
			if !owned {
				continue
			}
			var id int64
			if err := requiredRuleJSON(item, "id", &id); err != nil {
				return settings.RulesetObservation{}, err
			}
			if id <= 0 || seen[id] {
				return settings.RulesetObservation{}, fmt.Errorf("invalid/duplicate Ruleset ID")
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if len(items) < 100 {
			break
		}
	}
	values := []settings.RulesetState{}
	push := 0
	for _, id := range ids {
		var raw map[string]json.RawMessage
		if err := client.rest.Get(fmt.Sprintf("%s/%d?includes_parents=false", rulesetsPath(repo), id), &raw); err != nil {
			return settings.RulesetObservation{}, fmt.Errorf("read Ruleset %d: %w", id, err)
		}
		owned, err := rulesetOwner(raw, repo)
		if err != nil {
			return settings.RulesetObservation{}, err
		}
		if !owned {
			return settings.RulesetObservation{}, fmt.Errorf("Ruleset ownership changed during read")
		}
		var actual int64
		var target string
		if err := requiredRuleJSON(raw, "id", &actual); err != nil {
			return settings.RulesetObservation{}, err
		}
		if actual != id {
			return settings.RulesetObservation{}, fmt.Errorf("Ruleset ID changed during read")
		}
		if err := requiredRuleJSON(raw, "target", &target); err != nil {
			return settings.RulesetObservation{}, err
		}
		if target == "push" {
			push++
			continue
		}
		if target != "branch" && target != "tag" {
			return settings.RulesetObservation{}, fmt.Errorf("unsupported Ruleset target %q", target)
		}
		policy, err := decodeRulesetPolicy(raw)
		if err != nil {
			return settings.RulesetObservation{}, fmt.Errorf("read Ruleset %d: %w", id, err)
		}
		values = append(values, settings.RulesetState{ID: id, Policy: policy})
	}
	return settings.NewRulesetObservation(values, push)
}
func decodeRulesetPolicy(raw map[string]json.RawMessage) (settings.RulesetDeclaration, error) {
	policy := map[string]json.RawMessage{}
	for key, value := range raw {
		switch key {
		case "name", "target", "enforcement", "conditions", "bypass_actors", "rules":
			policy[key] = value
		case "id", "node_id", "source_type", "source", "_links", "created_at", "updated_at", "current_user_can_bypass": // Explicit read-only metadata.
		default:
			return settings.RulesetDeclaration{}, fmt.Errorf("unknown Ruleset response field %q", key)
		}
	}
	// The API uses ignored/null actor IDs for actors without numeric identity.
	var actors []map[string]json.RawMessage
	if err := requiredRuleJSON(policy, "bypass_actors", &actors); err != nil {
		return settings.RulesetDeclaration{}, fmt.Errorf("complete bypass visibility required: %w", err)
	}
	if actors == nil {
		return settings.RulesetDeclaration{}, fmt.Errorf("null bypass actors")
	}
	for _, a := range actors {
		var kind string
		if err := requiredRuleJSON(a, "actor_type", &kind); err != nil {
			return settings.RulesetDeclaration{}, err
		}
		if kind == "OrganizationAdmin" || kind == "DeployKey" {
			if id, ok := a["actor_id"]; ok && !bytes.Equal(bytes.TrimSpace(id), []byte("null")) {
				var ignored int64
				if err := json.Unmarshal(id, &ignored); err != nil {
					return settings.RulesetDeclaration{}, err
				}
				if kind == "DeployKey" {
					return settings.RulesetDeclaration{}, fmt.Errorf("DeployKey actor_id must be null")
				}
			}
			delete(a, "actor_id")
		}
	}
	encoded, err := json.Marshal(actors)
	if err != nil {
		return settings.RulesetDeclaration{}, err
	}
	policy["bypass_actors"] = encoded
	// Nullable integration_id is GitHub's any-source check identity. YAML retains omission,
	// never null. Reject null containers/parameters; do not synthesize policy defaults.
	var rules []map[string]json.RawMessage
	if err := requiredRuleJSON(policy, "rules", &rules); err != nil {
		return settings.RulesetDeclaration{}, err
	}
	for _, r := range rules {
		if v, ok := r["parameters"]; ok && bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return settings.RulesetDeclaration{}, fmt.Errorf("null rule parameters")
		}
		var kind string
		if err := requiredRuleJSON(r, "type", &kind); err != nil {
			return settings.RulesetDeclaration{}, err
		}
		if kind != "required_status_checks" {
			continue
		}
		if raw, ok := r["parameters"]; ok {
			var params map[string]json.RawMessage
			if err := json.Unmarshal(raw, &params); err != nil {
				return settings.RulesetDeclaration{}, err
			}
			if raw, ok := params["required_status_checks"]; ok {
				var checks []map[string]json.RawMessage
				if err := json.Unmarshal(raw, &checks); err != nil {
					return settings.RulesetDeclaration{}, err
				}
				if checks == nil {
					return settings.RulesetDeclaration{}, fmt.Errorf("null required status checks")
				}
				for _, c := range checks {
					if raw, ok := c["integration_id"]; ok && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
						delete(c, "integration_id")
					}
				}
				encoded, err := json.Marshal(checks)
				if err != nil {
					return settings.RulesetDeclaration{}, err
				}
				params["required_status_checks"] = encoded
			}
			encoded, err := json.Marshal(params)
			if err != nil {
				return settings.RulesetDeclaration{}, err
			}
			r["parameters"] = encoded
		}
	}
	encodedRules, err := json.Marshal(rules)
	if err != nil {
		return settings.RulesetDeclaration{}, err
	}
	policy["rules"] = encodedRules

	data, err := json.Marshal(policy)
	if err != nil {
		return settings.RulesetDeclaration{}, err
	}
	var keys any
	if err := json.Unmarshal(data, &keys); err != nil {
		return settings.RulesetDeclaration{}, err
	}
	if err := checkRulesetResponseKeys(keys, "policy"); err != nil {
		return settings.RulesetDeclaration{}, err
	}
	var result settings.RulesetDeclaration
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return settings.RulesetDeclaration{}, err
	}
	if err := settings.ValidateObservedRuleset(result); err != nil {
		return settings.RulesetDeclaration{}, err
	}
	return result, nil
}
func (client *Client) ApplyRuleset(repo repository.Repository, change settings.RulesetChange) (int64, error) {
	if change.Operation() != "create" && change.Operation() != "update" {
		return 0, fmt.Errorf("invalid Ruleset operation")
	}
	after := change.After()
	if after == nil {
		return 0, fmt.Errorf("missing Ruleset payload")
	}
	// YAML identity selects the URL resource, never a writable payload field.
	if after.ID != nil {
		if change.Operation() != "update" || *after.ID != change.ID() {
			return 0, fmt.Errorf("Ruleset declaration ID does not match update identity")
		}
		after.ID = nil
	}
	if err := settings.ValidateRuleset(*after, true); err != nil {
		return 0, err
	}
	data, err := json.Marshal(after)
	if err != nil {
		return 0, err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0, err
	}
	var actors []map[string]json.RawMessage
	if err := json.Unmarshal(payload["bypass_actors"], &actors); err != nil {
		return 0, err
	}
	for _, actor := range actors {
		if string(actor["actor_type"]) == `"DeployKey"` {
			actor["actor_id"] = json.RawMessage("null")
		}
	}
	payload["bypass_actors"], err = json.Marshal(actors)
	if err != nil {
		return 0, err
	}
	data, err = json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	path := rulesetsPath(repo)
	var response map[string]json.RawMessage
	if change.Operation() == "create" {
		err = client.rest.Post(path, bytes.NewReader(data), &response)
	} else {
		err = client.rest.Put(path+"/"+strconv.FormatInt(change.ID(), 10), bytes.NewReader(data), &response)
	}
	if err != nil {
		return 0, err
	}
	// Retain a reported positive creation ID even if subsequent verification fails;
	// the application must report uncertainty without retrying or writing it to YAML.
	var id int64
	if err := requiredRuleJSON(response, "id", &id); err != nil || id <= 0 {
		return 0, fmt.Errorf("verify Ruleset response: missing positive ID (writes may already have applied)")
	}
	fail := func(err error) (int64, error) {
		return id, fmt.Errorf("verify Ruleset response (repository writes may already have applied): %w", err)
	}
	if change.Operation() == "update" && id != change.ID() {
		return fail(fmt.Errorf("Ruleset ID differs from the selected update ID"))
	}
	owned, err := rulesetOwner(response, repo)
	if err != nil {
		return fail(err)
	}
	if !owned {
		return fail(fmt.Errorf("Ruleset response is not owned by the selected repository"))
	}
	if err := verifyReviewRulesetResponse(*after, response); err != nil {
		return fail(err)
	}
	return id, nil
}

// Verify the selected whole supported policy without running create/update
// selection again: a successful create response is not a name-adoption request.
func verifyReviewRulesetResponse(desired settings.RulesetDeclaration, raw map[string]json.RawMessage) error {
	policy, err := decodeRulesetPolicy(raw)
	if err != nil {
		return err
	}
	observed, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 1, Policy: policy}}, 0)
	if err != nil {
		return err
	}
	declaration, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{desired})
	if err != nil {
		return err
	}
	values, err := observed.Values()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(values[0].Policy, declaration.Values()[0]) {
		return fmt.Errorf("GitHub response did not retain the requested Ruleset policy")
	}
	return nil
}

// Check exact JSON keys before Go's case-insensitive struct-field decoder.
func checkRulesetResponseKeys(value any, context string) error {
	switch node := value.(type) {
	case nil:
		return fmt.Errorf("null Ruleset response value")
	case []any:
		for _, v := range node {
			if err := checkRulesetResponseKeys(v, context); err != nil {
				return err
			}
		}
	case map[string]any:
		var allowed []string
		switch context {
		case "policy":
			allowed = []string{"name", "target", "enforcement", "conditions", "bypass_actors", "rules"}
		case "conditions":
			allowed = []string{"ref_name"}
		case "ref_name":
			allowed = []string{"include", "exclude"}
		case "actor":
			allowed = []string{"actor_type", "actor_id", "bypass_mode"}
		case "rule":
			allowed = []string{"type", "parameters"}
		case "parameters":
			allowed = []string{"allowed_merge_methods", "dismiss_stale_reviews_on_push", "dismissal_restriction", "require_code_owner_review", "require_extra_approval_for_unattributed_changes", "require_last_push_approval", "required_approving_review_count", "required_review_thread_resolution", "required_reviewers", "required_status_checks", "strict_required_status_checks_policy", "do_not_enforce_on_create"}
		case "dismissal_restriction":
			allowed = []string{"enabled", "allowed_actors"}
		case "review_actor", "reviewer":
			allowed = []string{"id", "type"}
		case "required_reviewer":
			allowed = []string{"reviewer", "minimum_approvals", "file_patterns"}
		case "status_check":
			allowed = []string{"context", "integration_id"}
		}
		for key, v := range node {
			known := false
			for _, a := range allowed {
				if key == a {
					known = true
					break
				}
			}
			if !known {
				return fmt.Errorf("unknown Ruleset %s field %q", context, key)
			}
			next := key
			if key == "bypass_actors" {
				next = "actor"
			}
			if key == "rules" {
				next = "rule"
			}
			if key == "required_status_checks" {
				next = "status_check"
			}
			if key == "required_reviewers" {
				next = "required_reviewer"
			}
			if key == "allowed_actors" {
				next = "review_actor"
			}
			if err := checkRulesetResponseKeys(v, next); err != nil {
				return err
			}
		}
	}
	return nil
}
