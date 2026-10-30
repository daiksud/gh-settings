package presentation

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestTeamApprovalsPresentationUsesNumericIDOrderAndRealRemoval(t *testing.T) {
	for _, mode := range []string{"equal", "different", "reviewer removed", "replaced", "rule removed"} {
		for _, reverse := range []bool{false, true} {
			var before, after settings.RulesetDeclaration
			for _, p := range []*settings.RulesetDeclaration{&before, &after} {
				if err := json.Unmarshal([]byte(approvalsDisplayPolicy), p); err != nil {
					t.Fatal(err)
				}
				zero := 0
				patterns := []string{}
				(*p.Rules)[0].Parameters.RequiredReviewers = &[]settings.RequiredReviewer{
					{Reviewer: settings.ReviewActor{Type: "Team", ID: 2}, MinimumApprovals: &zero, FilePatterns: &patterns},
					{Reviewer: settings.ReviewActor{Type: "Team", ID: 10}, MinimumApprovals: &zero, FilePatterns: &patterns},
				}
			}
			reviewers := (*after.Rules)[0].Parameters.RequiredReviewers
			switch mode {
			case "different":
				ten := 10
				(*reviewers)[1].MinimumApprovals = &ten
			case "reviewer removed":
				*reviewers = (*reviewers)[:1]
			case "replaced":
				(*reviewers)[1].Reviewer.ID = 99
			case "rule removed":
				after.Rules = &[]settings.RulesetRule{}
			}
			records := settings.RulesetNormalizations{RequiredApprovals: settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: 11, Fallback: 0}, TeamApprovals: []settings.RulesetTeamApprovalsNormalization{
				{ReviewerID: 10, Path: settings.RulesetTeamApprovalsPath, RawValue: 11, Fallback: 0},
				{ReviewerID: 2, Path: settings.RulesetTeamApprovalsPath, RawValue: -1, Fallback: 0},
			}}
			if reverse {
				records.TeamApprovals[0], records.TeamApprovals[1] = records.TeamApprovals[1], records.TeamApprovals[0]
			}
			change, err := settings.NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records)
			if err != nil {
				t.Fatal(err)
			}
			var expected []string
			for _, id := range []int64{2, 10} {
				raw := 11
				if id == 2 {
					raw = -1
				}
				warning := `Warning: repository.settings.rulesets.rules.require_a_pull_request_before_merging.require_review_from_specific_teams.approvals: observed ` + strconv.Itoa(raw) + ` is outside UI choices; using 0 (projection only). Ruleset 7 "policy", reviewer=` + strconv.FormatInt(id, 10) + `.`
				desired := "0 (write required)"
				if mode == "rule removed" {
					desired = "inactive (rule removed; write required)"
				} else if id == 10 {
					switch mode {
					case "different":
						desired = "10 (write required)"
					case "reviewer removed", "replaced":
						desired = "inactive (reviewer removed; write required)"
					}
				}
				decision := `repository.settings.rulesets.rules.require_a_pull_request_before_merging.require_review_from_specific_teams.approvals (Ruleset 7 "policy", reviewer=` + strconv.FormatInt(id, 10) + `): 0 (normalized observation) → ` + desired
				expected = append(expected, warning, decision)
			}
			got := RulesetLine(change)
			if !strings.Contains(got, strings.Join(expected, "\n")+"\nEffective policy:") || strings.Count(got, "Effective policy:") != 1 || !strings.Contains(got, "required_approvals: observed 11") || strings.Index(got, "required_approvals: observed") > strings.Index(got, "require_review_from_specific_teams.approvals:") {
				t.Fatal(got)
			}
		}
	}
}
