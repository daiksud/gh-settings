package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIRulesetTeams(teams *[]uimodel.TeamReview) (*[]settings.RequiredReviewer, error) {
	if teams == nil {
		return nil, nil
	}
	result := make([]settings.RequiredReviewer, 0, len(*teams))
	for _, team := range *teams {
		if team.Approvals == nil || team.FilePatterns == nil {
			return nil, fmt.Errorf("team reviewer requires approvals and file_patterns")
		}
		if *team.Approvals < 0 || *team.Approvals > 10 {
			return nil, fmt.Errorf("team reviewer approvals must be a UI choice from 0 to 10")
		}
		patterns := append([]string{}, (*team.FilePatterns)...)
		result = append(result, settings.RequiredReviewer{Reviewer: settings.ReviewActor{Type: "Team", ID: team.Reviewer}, MinimumApprovals: copyUIRulesetValue(team.Approvals), FilePatterns: &patterns})
	}
	return &result, nil
}

func observeUIRulesetTeams(teams *[]settings.RequiredReviewer) (*[]uimodel.TeamReview, error) {
	if teams == nil {
		return nil, fmt.Errorf("missing acquired team reviewers")
	}
	if len(*teams) == 0 {
		return nil, nil
	}
	result := make([]uimodel.TeamReview, 0, len(*teams))
	for _, team := range *teams {
		if team.Reviewer.Type != "Team" || team.MinimumApprovals == nil || team.FilePatterns == nil {
			return nil, fmt.Errorf("incomplete or unsupported acquired team reviewer")
		}
		patterns := append([]string{}, (*team.FilePatterns)...)
		result = append(result, uimodel.TeamReview{Reviewer: team.Reviewer.ID, Approvals: copyUIRulesetValue(team.MinimumApprovals), FilePatterns: &patterns})
	}
	return &result, nil
}
