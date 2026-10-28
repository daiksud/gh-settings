package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

var uiRulesetRoles = map[uimodel.BypassRole]int64{uimodel.RepositoryAdmin: 5, uimodel.Maintain: 2, uimodel.Write: 4}

func declareUIRulesetBypass(entries *[]uimodel.BypassEntry) (*[]settings.BypassActor, error) {
	if entries == nil {
		return nil, nil
	}
	result := make([]settings.BypassActor, 0, len(*entries))
	for _, entry := range *entries {
		selected := 0
		actor := settings.BypassActor{}
		if entry.Role != nil {
			selected++
			id, ok := uiRulesetRoles[*entry.Role]
			if !ok {
				return nil, fmt.Errorf("unsupported bypass role %q", *entry.Role)
			}
			actor.ActorType, actor.ActorID = "RepositoryRole", &id
		}
		for _, dynamic := range []struct {
			id   *int64
			kind string
		}{{entry.Team, "Team"}, {entry.App, "Integration"}, {entry.User, "User"}} {
			if dynamic.id != nil {
				selected++
				actor.ActorType, actor.ActorID = dynamic.kind, copyUIRulesetValue(dynamic.id)
			}
		}
		if entry.DeployKeys != nil {
			selected++
			if !*entry.DeployKeys {
				return nil, fmt.Errorf("deploy_keys must be selected when present")
			}
			actor.ActorType = "DeployKey"
		}
		if selected != 1 {
			return nil, fmt.Errorf("bypass entry requires exactly one actor selector")
		}
		if entry.BypassMode == nil {
			return nil, fmt.Errorf("bypass entry requires bypass_mode")
		}
		mode := string(*entry.BypassMode)
		switch *entry.BypassMode {
		case uimodel.Always, uimodel.Exempt:
		case uimodel.ForPullRequestsOnly:
			mode = "pull_request"
		default:
			return nil, fmt.Errorf("unsupported UI bypass_mode %q", mode)
		}
		actor.BypassMode = &mode
		result = append(result, actor)
	}
	return &result, nil
}

func observeUIRulesetBypass(actors []settings.BypassActor) (*[]uimodel.BypassEntry, error) {
	result := make([]uimodel.BypassEntry, 0, len(actors))
	for _, actor := range actors {
		entry := uimodel.BypassEntry{}
		switch actor.ActorType {
		case "RepositoryRole":
			if actor.ActorID == nil {
				return nil, fmt.Errorf("missing acquired bypass role")
			}
			for role, id := range uiRulesetRoles {
				if id == *actor.ActorID {
					entry.Role = copyUIRulesetValue(&role)
				}
			}
			if entry.Role == nil {
				return nil, fmt.Errorf("bypass role is outside UI choices")
			}
		case "Team":
			entry.Team = copyUIRulesetValue(actor.ActorID)
		case "Integration":
			entry.App = copyUIRulesetValue(actor.ActorID)
		case "User":
			entry.User = copyUIRulesetValue(actor.ActorID)
		case "DeployKey":
			selected := true
			entry.DeployKeys = &selected
		default:
			return nil, fmt.Errorf("bypass actor type is outside UI choices")
		}
		if actor.BypassMode == nil {
			return nil, fmt.Errorf("missing acquired bypass mode")
		}
		mode := uimodel.BypassMode(*actor.BypassMode)
		switch *actor.BypassMode {
		case "always", "exempt":
		case "pull_request":
			mode = uimodel.ForPullRequestsOnly
		default:
			return nil, fmt.Errorf("bypass mode is outside UI choices")
		}
		entry.BypassMode = &mode
		result = append(result, entry)
	}
	return &result, nil
}
