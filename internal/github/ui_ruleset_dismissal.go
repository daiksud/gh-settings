package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIRulesetDismissal(ui uimodel.PullRequestRule) (*settings.ReviewDismissalRestriction, error) {
	if ui.RestrictWhoCanDismissPullRequestReviews == nil && ui.PeopleTeamsOrAppsThatCanDismissReviews == nil {
		return nil, nil
	}
	if ui.PeopleTeamsOrAppsThatCanDismissReviews != nil && len(*ui.PeopleTeamsOrAppsThatCanDismissReviews) > 0 && (ui.RestrictWhoCanDismissPullRequestReviews == nil || !*ui.RestrictWhoCanDismissPullRequestReviews) {
		return nil, fmt.Errorf("nonempty review dismissal actors require an explicitly selected restriction checkbox")
	}
	result := &settings.ReviewDismissalRestriction{Enabled: copyUIRulesetValue(ui.RestrictWhoCanDismissPullRequestReviews)}
	if ui.PeopleTeamsOrAppsThatCanDismissReviews != nil {
		actors := make([]settings.ReviewActor, 0, len(*ui.PeopleTeamsOrAppsThatCanDismissReviews))
		for _, entry := range *ui.PeopleTeamsOrAppsThatCanDismissReviews {
			selected := 0
			actor := settings.ReviewActor{}
			if entry.Role != nil {
				selected++
				id, ok := uiRulesetRoles[*entry.Role]
				if !ok {
					return nil, fmt.Errorf("unsupported review dismissal role")
				}
				actor.Type, actor.ID = "RepositoryRole", id
			}
			for _, dynamic := range []struct {
				id   *int64
				kind string
			}{{entry.Team, "Team"}, {entry.User, "User"}, {entry.App, "IntegrationInstallation"}} {
				if dynamic.id != nil {
					selected++
					actor.Type, actor.ID = dynamic.kind, *dynamic.id
				}
			}
			if selected != 1 {
				return nil, fmt.Errorf("review dismissal entry requires exactly one actor")
			}
			actors = append(actors, actor)
		}
		result.AllowedActors = &actors
	}
	return result, nil
}

func observeUIRulesetDismissal(raw *settings.ReviewDismissalRestriction) (*bool, *[]uimodel.ReviewDismissalActor, error) {
	if raw == nil {
		return nil, nil, nil
	}
	if raw.Enabled == nil || raw.AllowedActors == nil {
		return nil, nil, fmt.Errorf("incomplete acquired review dismissal restriction")
	}
	actors := make([]uimodel.ReviewDismissalActor, 0, len(*raw.AllowedActors))
	for _, actor := range *raw.AllowedActors {
		entry := uimodel.ReviewDismissalActor{}
		switch actor.Type {
		case "RepositoryRole":
			for role, id := range uiRulesetRoles {
				if actor.ID == id {
					entry.Role = copyUIRulesetValue(&role)
				}
			}
			if entry.Role == nil {
				return nil, nil, fmt.Errorf("review dismissal role is outside UI choices")
			}
		case "User":
			entry.User = copyUIRulesetValue(&actor.ID)
		case "Team":
			entry.Team = copyUIRulesetValue(&actor.ID)
		case "IntegrationInstallation":
			entry.App = copyUIRulesetValue(&actor.ID)
		default:
			return nil, nil, fmt.Errorf("review dismissal actor is outside UI identities")
		}
		actors = append(actors, entry)
	}
	if !*raw.Enabled {
		return copyUIRulesetValue(raw.Enabled), nil, nil
	}
	return copyUIRulesetValue(raw.Enabled), &actors, nil
}
