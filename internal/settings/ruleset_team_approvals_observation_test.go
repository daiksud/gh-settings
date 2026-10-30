package settings

import (
	"reflect"
	"testing"
)

func normalizedTeamApprovalsState(companions bool) RulesetState {
	state := normalizedMergeMethodsState([]string{"METHOD_FUTURE"}, companions)
	state.Policy.Name = "teams"
	if !companions {
		state.MergeMethodsNormalization = RulesetMergeMethodsNormalization{}
	}
	ruleOfForApprovals(state.Policy).RequiredReviewers = &[]RequiredReviewer{
		{Reviewer: ReviewActor{Type: "Team", ID: 10}, MinimumApprovals: paramInt(0), FilePatterns: rsList("**", "!generated/**", "**")},
		{Reviewer: ReviewActor{Type: "Team", ID: 2}, MinimumApprovals: paramInt(0), FilePatterns: rsList("docs/**")},
		{Reviewer: ReviewActor{Type: "Team", ID: 5}, MinimumApprovals: paramInt(3), FilePatterns: rsList()},
	}
	state.TeamApprovalsNormalizations = []RulesetTeamApprovalsNormalization{
		{ReviewerID: 10, Path: RulesetTeamApprovalsPath, RawValue: 11, Fallback: 0},
		{ReviewerID: 2, Path: RulesetTeamApprovalsPath, RawValue: -1, Fallback: 0},
	}
	return state
}

func TestTeamApprovalsMappingAndOwnedObservation(t *testing.T) {
	if RulesetTeamApprovalsPath != "repository.settings.rulesets.rules.require_a_pull_request_before_merging.require_review_from_specific_teams.approvals" || DefaultRulesetTeamApprovals != 0 {
		t.Fatal("path/default")
	}
	for _, value := range []int{-1, 0, 1, 10, 11, 100} {
		supported := value >= 0 && value <= 10
		if RulesetTeamApprovalsSupported(value) != supported {
			t.Fatal(value)
		}
		policy := reviewPolicy()
		*(*ruleOf(t, policy, "pull_request").RequiredReviewers)[0].MinimumApprovals = value
		if (ValidateRuleset(policy, true) == nil) != supported {
			t.Fatal("native validation changed", value)
		}
	}
	for _, companions := range []bool{false, true} {
		state := normalizedTeamApprovalsState(companions)
		observed, err := NewRulesetObservation([]RulesetState{state}, 0)
		if err != nil {
			t.Fatal(err)
		}
		state.TeamApprovalsNormalizations[0].ReviewerID = 99
		*(*ruleOf(t, state.Policy, "pull_request").RequiredReviewers)[0].MinimumApprovals = 9
		(*(*ruleOf(t, state.Policy, "pull_request").RequiredReviewers)[0].FilePatterns)[0] = "CORRUPT"
		values, err := observed.Values()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := values[0].RawRequiredReviewers()
		if err != nil || len(raw) != 3 || raw[0].Reviewer.ID != 2 || *raw[0].MinimumApprovals != -1 || raw[1].Reviewer.ID != 5 || *raw[1].MinimumApprovals != 3 || raw[1].FilePatterns == nil || len(*raw[1].FilePatterns) != 0 || raw[2].Reviewer.ID != 10 || *raw[2].MinimumApprovals != 11 || !reflect.DeepEqual(*raw[2].FilePatterns, []string{"**", "!generated/**", "**"}) {
			t.Fatal("raw reviewer provenance lost", raw, err)
		}
		*raw[0].MinimumApprovals = 9
		(*raw[2].FilePatterns)[0] = "CORRUPT"
		raw[2].Reviewer.ID = 99
		again, err := values[0].RawRequiredReviewers()
		if err != nil || *again[0].MinimumApprovals != -1 || again[2].Reviewer.ID != 10 || (*again[2].FilePatterns)[0] != "**" {
			t.Fatal("raw getter aliases state", again, err)
		}
		records := OrderedRulesetTeamApprovalsNormalizations(values[0].TeamApprovalsNormalizations)
		if len(records) != 2 || records[0].ReviewerID != 2 || records[0].RawValue != -1 || records[1].ReviewerID != 10 || records[1].RawValue != 11 {
			t.Fatal(records)
		}
		records[0].ReviewerID = 99
		values[0].TeamApprovalsNormalizations[0].RawValue = 999
		fresh, _ := observed.Values()
		if !reflect.DeepEqual(fresh[0].TeamApprovalsNormalizations, normalizedTeamApprovalsState(companions).TeamApprovalsNormalizations) {
			t.Fatal("evidence alias")
		}
		exported, err := observed.ExportDeclarations()
		if err != nil {
			t.Fatal(err)
		}
		reviewers := *ruleOf(t, exported.Values()[0], "pull_request").RequiredReviewers
		if *reviewers[0].MinimumApprovals != 0 || *reviewers[1].MinimumApprovals != 3 || *reviewers[2].MinimumApprovals != 0 {
			t.Fatal("export used raw counts", reviewers)
		}
		if companions && (fresh[0].EnforcementNormalization.RawValue != "evaluate" || fresh[0].RequiredApprovalsNormalization.RawValue != 11 || fresh[0].MergeMethodsNormalization.IsZero() || len(fresh[0].BypassModeNormalizations) == 0) {
			t.Fatal("companions lost")
		}
	}
}

func TestTeamApprovalsObservationRejectsForgedOrUnboundEvidence(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for name, mutate := range map[string]func(*RulesetState){
			"invalid ID":           func(s *RulesetState) { s.ID = 0 },
			"conflicting ID":       func(s *RulesetState) { s.Policy.ID = rsID(9) },
			"missing reviewer":     func(s *RulesetState) { s.TeamApprovalsNormalizations[0].ReviewerID = 99 },
			"zero reviewer":        func(s *RulesetState) { s.TeamApprovalsNormalizations[0].ReviewerID = 0 },
			"wrong path":           func(s *RulesetState) { s.TeamApprovalsNormalizations[0].Path = "other" },
			"canonical raw":        func(s *RulesetState) { s.TeamApprovalsNormalizations[0].RawValue = 0 },
			"wrong fallback":       func(s *RulesetState) { s.TeamApprovalsNormalizations[0].Fallback = 1 },
			"wrong effective":      func(s *RulesetState) { *(*ruleOfForApprovals(s.Policy).RequiredReviewers)[0].MinimumApprovals = 1 },
			"unsupported identity": func(s *RulesetState) { (*ruleOfForApprovals(s.Policy).RequiredReviewers)[0].Reviewer.Type = "User" },
			"missing patterns":     func(s *RulesetState) { (*ruleOfForApprovals(s.Policy).RequiredReviewers)[0].FilePatterns = nil },
			"duplicate reviewer":   func(s *RulesetState) { v := ruleOfForApprovals(s.Policy).RequiredReviewers; *v = append(*v, (*v)[0]) },
			"too many reviewers": func(s *RulesetState) {
				v := ruleOfForApprovals(s.Policy).RequiredReviewers
				for i := int64(100); i < 113; i++ {
					*v = append(*v, RequiredReviewer{Reviewer: ReviewActor{Type: "Team", ID: i}, MinimumApprovals: paramInt(0), FilePatterns: rsList()})
				}
			},
			"invalid companion": func(s *RulesetState) {
				s.MergeMethodsNormalization = RulesetMergeMethodsNormalization{RawValue: []string{}}
			},
			"duplicate evidence": func(s *RulesetState) {
				s.TeamApprovalsNormalizations = append(s.TeamApprovalsNormalizations, s.TeamApprovalsNormalizations[0])
			},
			"empty evidence":  func(s *RulesetState) { s.TeamApprovalsNormalizations[0] = RulesetTeamApprovalsNormalization{} },
			"empty reviewers": func(s *RulesetState) { ruleOfForApprovals(s.Policy).RequiredReviewers = &[]RequiredReviewer{} },
			"absent PR":       func(s *RulesetState) { s.Policy.Rules = &[]RulesetRule{} },
		} {
			t.Run(name, func(t *testing.T) {
				state := normalizedTeamApprovalsState(companions)
				mutate(&state)
				if got, err := state.RawRequiredReviewers(); err == nil || got != nil {
					t.Fatal("invalid raw view", got, err)
				}
				ordinary := RulesetState{ID: 8, Policy: reviewPolicy()}
				if got, err := NewRulesetObservation([]RulesetState{ordinary, state}, 0); err == nil || !reflect.DeepEqual(got, RulesetObservation{}) {
					t.Fatal("invalid partial observation", got, err)
				}
			})
		}
	}
}

func TestTeamApprovalsObservationKeepsAbsenceAndEmptyReviewers(t *testing.T) {
	for _, absent := range []bool{false, true} {
		state := RulesetState{ID: 7, Policy: reviewPolicy()}
		ruleOfForApprovals(state.Policy).RequiredReviewers = &[]RequiredReviewer{}
		if absent {
			state.Policy.Rules = &[]RulesetRule{}
		}
		got, err := state.RawRequiredReviewers()
		if err != nil || len(got) != 0 || (got == nil) != absent {
			t.Fatal(got, err)
		}
	}
	if got, err := (RulesetState{}).RawRequiredReviewers(); err == nil || got != nil {
		t.Fatal("unacquired became empty")
	}
}
