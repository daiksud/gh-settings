package workspace

import (
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

func parseActionsPolicy(node *yaml.Node) (settings.ActionsPolicyDeclaration, error) {
	var d settings.ActionsPolicyDeclaration
	fields, err := loadMapping(node, "settings.actions")
	if err != nil {
		return d, err
	}
	for field, node := range fields {
		n, err := resolveAlias(&node)
		if err != nil {
			return d, err
		}
		switch field {
		case "enabled", "sha_pinning_required":
			v, err := actionsYAMLBool(n)
			if err != nil {
				return d, err
			}
			if field == "enabled" {
				d.Enabled = &v
			} else {
				d.SHAPinningRequired = &v
			}
		case "allowed_actions":
			if n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
				return d, fmt.Errorf("actions.allowed_actions must be a string")
			}
			value := n.Value
			d.AllowedActions = &value
		case "selected_actions":
			selected, err := loadMapping(n, "actions.selected_actions")
			if err != nil {
				return d, err
			}
			for key, value := range selected {
				n, err := resolveAlias(&value)
				if err != nil {
					return d, err
				}
				switch key {
				case "github_owned_allowed", "verified_allowed":
					v, err := actionsYAMLBool(n)
					if err != nil {
						return d, err
					}
					if key == "github_owned_allowed" {
						d.SelectedActions.GitHubOwnedAllowed = &v
					} else {
						d.SelectedActions.VerifiedAllowed = &v
					}
				case "patterns_allowed":
					if n.Kind != yaml.SequenceNode {
						return d, fmt.Errorf("actions.selected_actions.patterns_allowed must be an array")
					}
					values := []string{}
					for _, item := range n.Content {
						v, err := resolveAlias(item)
						if err != nil {
							return d, err
						}
						if v.Kind != yaml.ScalarNode || v.Tag != "!!str" {
							return d, fmt.Errorf("Actions patterns must be strings")
						}
						values = append(values, v.Value)
					}
					d.SelectedActions.PatternsAllowed = &values
				default:
					return d, fmt.Errorf("unknown actions.selected_actions key %q", key)
				}
			}
		default:
			return d, fmt.Errorf("unknown settings.actions key %q", field)
		}
	}
	return d, d.Validate()
}

func actionsYAMLBool(n *yaml.Node) (bool, error) {
	if n.Kind != yaml.ScalarNode || n.Tag != "!!bool" || (!strings.EqualFold(n.Value, "true") && !strings.EqualFold(n.Value, "false")) {
		return false, fmt.Errorf("Actions boolean values must be spelled true or false")
	}
	return strings.EqualFold(n.Value, "true"), nil
}
