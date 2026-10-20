package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

func parseRulesets(node *yaml.Node) (*settings.RulesetDeclarations, error) {
	value, err := rulesetYAMLValue(node, "policy", 0)
	if err != nil {
		return nil, err
	}
	if _, ok := value.([]any); !ok {
		return nil, fmt.Errorf("settings.rulesets must be an array of mappings")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var entries []settings.RulesetDeclaration
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode rulesets: %w", err)
	}
	result, err := settings.NewRulesetDeclarations(entries)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Convert resolved YAML nodes with exact scalar types, retaining empty collections.
// Strict JSON decoding below checks keys and field shapes against the typed schema.
func rulesetYAMLValue(node *yaml.Node, context string, depth int) (any, error) {
	if depth > 100 {
		return nil, fmt.Errorf("ruleset YAML nesting or alias cycle exceeds limit")
	}
	n, err := resolveAlias(node)
	if err != nil {
		return nil, err
	}
	switch n.Kind {
	case yaml.MappingNode:
		fields, err := loadMapping(n, "ruleset mapping")
		if err != nil {
			return nil, err
		}
		out := map[string]any{}
		for k, v := range fields {
			if !rulesetYAMLKey(context, k) {
				return nil, fmt.Errorf("unknown ruleset %s field %q", context, k)
			}
			next := k
			if k == "rules" {
				next = "rule"
			}
			if k == "bypass_actors" {
				next = "actor"
			}
			value, err := rulesetYAMLValue(&v, next, depth+1)
			if err != nil {
				return nil, err
			}
			out[k] = value
		}
		return out, nil
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, v := range n.Content {
			value, err := rulesetYAMLValue(v, context, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, value)
		}
		return out, nil
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return n.Value, nil
		case "!!int":
			var i int64
			if err := n.Decode(&i); err != nil {
				return nil, err
			}
			return i, nil
		case "!!bool":
			if n.Value != "true" && n.Value != "false" {
				return nil, fmt.Errorf("ruleset boolean must be true or false")
			}
			return n.Value == "true", nil
		}
	}
	return nil, fmt.Errorf("null or unsupported ruleset YAML value")
}

func rulesetYAMLKey(context, key string) bool {
	var allowed []string
	switch context {
	case "policy":
		allowed = []string{"name", "target", "enforcement", "conditions", "bypass_actors", "rules"}
	case "conditions":
		allowed = []string{"ref_name"}
	case "ref_name":
		allowed = []string{"include", "exclude"}
	case "rule":
		allowed = []string{"type", "parameters"}
	case "actor":
		allowed = []string{"actor_type", "actor_id", "bypass_mode"}
	}
	for _, v := range allowed {
		if v == key {
			return true
		}
	}
	return false
}
