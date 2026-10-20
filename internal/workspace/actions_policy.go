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
		case "enabled", "sha_pinning_required", "can_approve_pull_request_reviews":
			v, err := actionsYAMLBool(n)
			if err != nil {
				return d, err
			}
			if field == "can_approve_pull_request_reviews" {
				d.CanApprovePullRequestReviews = &v
			} else if field == "enabled" {
				d.Enabled = &v
			} else {
				d.SHAPinningRequired = &v
			}
		case "artifact_and_log_retention_days":
			if n.Kind != yaml.ScalarNode || n.Tag != "!!int" {
				return d, fmt.Errorf("actions.artifact_and_log_retention_days must be an integer")
			}
			var days int
			if err := n.Decode(&days); err != nil {
				return d, fmt.Errorf("invalid actions.artifact_and_log_retention_days: %w", err)
			}
			d.RetentionDays = &days
		case "fork_pr_approval_policy":
			if n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
				return d, fmt.Errorf("actions.fork_pr_approval_policy must be a string")
			}
			v := n.Value
			d.ForkApprovalPolicy = &v
		case "private_fork_workflows":
			v, err := parsePrivateFork(n)
			if err != nil {
				return d, err
			}
			d.PrivateFork = v
		case "default_workflow_permissions":
			if n.Kind != yaml.ScalarNode || n.Tag != "!!str" {
				return d, fmt.Errorf("actions.default_workflow_permissions must be a string")
			}
			value := n.Value
			d.DefaultWorkflowPermissions = &value
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

func parsePrivateFork(n *yaml.Node) (settings.PrivateForkDeclaration, error) {
	var d settings.PrivateForkDeclaration
	fields, err := loadMapping(n, "actions.private_fork_workflows")
	if err != nil {
		return d, err
	}
	for key, node := range fields {
		n, err := resolveAlias(&node)
		if err != nil {
			return d, err
		}
		v, err := actionsYAMLBool(n)
		if err != nil {
			return d, err
		}
		switch key {
		case "run_workflows_from_fork_pull_requests":
			d.Run = &v
		case "send_write_tokens_to_workflows":
			d.WriteTokens = &v
		case "send_secrets_and_variables":
			d.Secrets = &v
		case "require_approval_for_fork_pr_workflows":
			d.Approval = &v
		default:
			return d, fmt.Errorf("unknown actions.private_fork_workflows key %q", key)
		}
	}
	return d, nil
}
