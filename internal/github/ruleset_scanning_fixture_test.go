package github

import "encoding/json"

func scanningReadBaseBody(_ int, companions bool) map[string]any {
	var body map[string]any
	json.Unmarshal([]byte(fullReviewRulesetJSON), &body)
	if companions {
		body["enforcement"] = "FUTURE_ENFORCEMENT"
		for _, value := range body["rules"].([]any) {
			rule := value.(map[string]any)
			if rule["type"] == "pull_request" {
				p := rule["parameters"].(map[string]any)
				p["required_approving_review_count"] = 11
				p["allowed_merge_methods"] = []any{"FUTURE_METHOD"}
			}
		}
		body["bypass_actors"] = []any{map[string]any{"actor_type": "Team", "actor_id": 9, "bypass_mode": "FUTURE_MODE"}}
	}
	return body
}
