package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

func parseEnvironments(node *yaml.Node) (*settings.EnvironmentDeclarations, error) {
	node, err := resolveAlias(node)
	if err != nil {
		return nil, err
	}
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("settings.environments must be an array of mappings")
	}
	entries := []settings.EnvironmentDeclaration{}
	for _, entry := range node.Content {
		value, err := environmentYAMLValue(entry, "environment", 0)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		var policy settings.EnvironmentDeclaration
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&policy); err != nil {
			return nil, fmt.Errorf("decode environment: %w", err)
		}
		entries = append(entries, policy)
	}
	result, err := settings.NewEnvironmentDeclarations(entries)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Null is meaningful only for deployment_branch_policy (all branches/tags).
// YAML types remain exact; null must not silently acquire declaration defaults.
func environmentYAMLValue(node *yaml.Node, context string, depth int) (any, error) {
	if depth > 100 {
		return nil, fmt.Errorf("environment YAML nesting exceeds limit")
	}
	n, err := resolveAlias(node)
	if err != nil {
		return nil, err
	}
	if context == "deployment_branch_policy" && n.Kind == yaml.ScalarNode && n.Tag == "!!null" {
		return nil, nil
	}
	if context == "reviewers" || context == "deployment_branch_patterns" {
		if n.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("environment %s must be an array", context)
		}
		values := []any{}
		for _, entry := range n.Content {
			kind := "reviewer"
			if context == "deployment_branch_patterns" {
				kind = "pattern"
			}
			v, err := environmentYAMLValue(entry, kind, depth+1)
			if err != nil {
				return nil, err
			}
			values = append(values, v)
		}
		return values, nil
	}
	if context != "environment" && context != "reviewer" && context != "pattern" && context != "deployment_branch_policy" {
		if n.Kind != yaml.ScalarNode || n.Tag == "!!null" {
			return nil, fmt.Errorf("invalid environment %s value", context)
		}
		switch context {
		case "name", "type":
			if n.Tag != "!!str" {
				return nil, fmt.Errorf("environment %s must be a string", context)
			}
			return n.Value, nil
		case "id", "wait_timer":
			if n.Tag != "!!int" {
				return nil, fmt.Errorf("environment %s must be an integer", context)
			}
			var v int64
			if err := n.Decode(&v); err != nil {
				return nil, err
			}
			return v, nil
		default:
			if n.Tag != "!!bool" || (n.Value != "true" && n.Value != "false") {
				return nil, fmt.Errorf("environment %s must be true or false", context)
			}
			return n.Value == "true", nil
		}
	}
	fields, err := loadMapping(n, "environment "+context)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	for key, value := range fields {
		allowed := false
		switch context {
		case "environment":
			allowed = key == "name" || key == "wait_timer" || key == "prevent_self_review" || key == "can_admins_bypass" || key == "reviewers" || key == "deployment_branch_policy" || key == "deployment_branch_patterns"
		case "reviewer":
			allowed = key == "id" || key == "type"
		case "pattern":
			allowed = key == "name" || key == "type"
		case "deployment_branch_policy":
			allowed = key == "protected_branches" || key == "custom_branch_policies"
		}
		if !allowed {
			return nil, fmt.Errorf("unknown environment %s field %q", context, key)
		}
		v, err := environmentYAMLValue(&value, key, depth+1)
		if err != nil {
			return nil, err
		}
		result[key] = v
	}
	if context == "deployment_branch_policy" && len(result) != 2 {
		return nil, fmt.Errorf("environment deployment_branch_policy requires both booleans")
	}
	return result, nil
}
