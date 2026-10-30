package settings

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const RulesetTeamApprovalsPath = "repository.settings.rulesets.rules.require_a_pull_request_before_merging.require_review_from_specific_teams.approvals"
const DefaultRulesetTeamApprovals = uimodel.DefaultTeamReviewApprovals

func RulesetTeamApprovalsSupported(value int) bool { return value >= 0 && value <= 10 }

// RulesetTeamApprovalsNormalization binds an acquired count to one Team reviewer.
// Complete policy validation independently owns identity and pattern validity.
type RulesetTeamApprovalsNormalization struct {
	ReviewerID int64
	Path       string
	RawValue   int
	Fallback   int
}

func rulesetRequiredReviewers(policy RulesetDeclaration) *[]RequiredReviewer {
	if policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "pull_request" && rule.Parameters != nil {
				return rule.Parameters.RequiredReviewers
			}
		}
	}
	return nil
}

// RawRequiredReviewers returns owned original counts and patterns after validating
// all evidence. Nil means an absent PR rule; a present empty list remains empty.
func (state RulesetState) RawRequiredReviewers() ([]RequiredReviewer, error) {
	if err := state.validateNormalizationEvidence(); err != nil {
		return nil, err
	}
	reviewers := rulesetRequiredReviewers(cloneRuleset(state.Policy))
	if reviewers == nil {
		return nil, nil
	}
	for _, record := range state.TeamApprovalsNormalizations {
		for i := range *reviewers {
			if (*reviewers)[i].Reviewer.ID == record.ReviewerID {
				raw := record.RawValue
				(*reviewers)[i].MinimumApprovals = &raw
			}
		}
	}
	return *reviewers, nil
}

func (state RulesetState) validateTeamApprovalsNormalizations() error {
	reviewers := map[int64]RequiredReviewer{}
	if values := rulesetRequiredReviewers(state.Policy); values != nil {
		for _, reviewer := range *values {
			reviewers[reviewer.Reviewer.ID] = reviewer
		}
	}
	seen := map[int64]bool{}
	for _, record := range state.TeamApprovalsNormalizations {
		reviewer, found := reviewers[record.ReviewerID]
		if !found || seen[record.ReviewerID] || reviewer.Reviewer.Type != "Team" || *reviewer.MinimumApprovals != DefaultRulesetTeamApprovals || record.Path != RulesetTeamApprovalsPath || record.Fallback != DefaultRulesetTeamApprovals || RulesetTeamApprovalsSupported(record.RawValue) {
			return fmt.Errorf("invalid Ruleset team approvals observation provenance")
		}
		seen[record.ReviewerID] = true
	}
	return nil
}

// OrderedRulesetTeamApprovalsNormalizations returns independent records in numeric
// Team ID order without repairing or validating identities.
func OrderedRulesetTeamApprovalsNormalizations(records []RulesetTeamApprovalsNormalization) []RulesetTeamApprovalsNormalization {
	result := slices.Clone(records)
	slices.SortFunc(result, func(a, b RulesetTeamApprovalsNormalization) int { return cmp.Compare(a.ReviewerID, b.ReviewerID) })
	return result
}
