package settings

import (
	"fmt"
	"slices"
	"strings"
)

// RuleParameters is the closed parameter union selected by the surrounding rule type.
// Pointers retain omission independently of false, zero and empty collections.
type RuleParameters struct {
	AllowedMergeMethods                        *[]string                   `json:"allowed_merge_methods,omitempty" yaml:"allowed_merge_methods,omitempty"`
	DismissStaleReviewsOnPush                  *bool                       `json:"dismiss_stale_reviews_on_push,omitempty" yaml:"dismiss_stale_reviews_on_push,omitempty"`
	DismissalRestriction                       *ReviewDismissalRestriction `json:"dismissal_restriction,omitempty" yaml:"dismissal_restriction,omitempty"`
	RequireCodeOwnerReview                     *bool                       `json:"require_code_owner_review,omitempty" yaml:"require_code_owner_review,omitempty"`
	RequireExtraApprovalForUnattributedChanges *bool                       `json:"require_extra_approval_for_unattributed_changes,omitempty" yaml:"require_extra_approval_for_unattributed_changes,omitempty"`
	RequireLastPushApproval                    *bool                       `json:"require_last_push_approval,omitempty" yaml:"require_last_push_approval,omitempty"`
	RequiredApprovingReviewCount               *int                        `json:"required_approving_review_count,omitempty" yaml:"required_approving_review_count,omitempty"`
	RequiredReviewThreadResolution             *bool                       `json:"required_review_thread_resolution,omitempty" yaml:"required_review_thread_resolution,omitempty"`
	RequiredReviewers                          *[]RequiredReviewer         `json:"required_reviewers,omitempty" yaml:"required_reviewers,omitempty"`
	RequiredStatusChecks                       *[]RequiredStatusCheck      `json:"required_status_checks,omitempty" yaml:"required_status_checks,omitempty"`
	StrictRequiredStatusChecksPolicy           *bool                       `json:"strict_required_status_checks_policy,omitempty" yaml:"strict_required_status_checks_policy,omitempty"`
	DoNotEnforceOnCreate                       *bool                       `json:"do_not_enforce_on_create,omitempty" yaml:"do_not_enforce_on_create,omitempty"`
}
type ReviewActor struct {
	ID   int64  `json:"id" yaml:"id"`
	Type string `json:"type" yaml:"type"`
}
type ReviewDismissalRestriction struct {
	AllowedActors *[]ReviewActor `json:"allowed_actors,omitempty" yaml:"allowed_actors,omitempty"`
	Enabled       *bool          `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}
type RequiredReviewer struct {
	Reviewer         ReviewActor `json:"reviewer" yaml:"reviewer"`
	MinimumApprovals *int        `json:"minimum_approvals,omitempty" yaml:"minimum_approvals,omitempty"`
	FilePatterns     *[]string   `json:"file_patterns,omitempty" yaml:"file_patterns,omitempty"`
}
type RequiredStatusCheck struct {
	Context       string `json:"context" yaml:"context"`
	IntegrationID *int64 `json:"integration_id,omitempty" yaml:"integration_id,omitempty"`
}

func (p *RuleParameters) hasPR() bool {
	return p != nil && (p.AllowedMergeMethods != nil || p.DismissStaleReviewsOnPush != nil || p.DismissalRestriction != nil || p.RequireCodeOwnerReview != nil || p.RequireExtraApprovalForUnattributedChanges != nil || p.RequireLastPushApproval != nil || p.RequiredApprovingReviewCount != nil || p.RequiredReviewThreadResolution != nil || p.RequiredReviewers != nil)
}
func (p *RuleParameters) hasChecks() bool {
	return p != nil && (p.RequiredStatusChecks != nil || p.StrictRequiredStatusChecksPolicy != nil || p.DoNotEnforceOnCreate != nil)
}
func reviewActorKey(a ReviewActor) string { return fmt.Sprintf("%s:%d", a.Type, a.ID) }
func checkKey(c RequiredStatusCheck) string {
	if c.IntegrationID == nil {
		return c.Context + "\x00"
	}
	return fmt.Sprintf("%s\x00%d", c.Context, *c.IntegrationID)
}
func validateRule(r RulesetRule, complete bool) error {
	if !supportedRule(r.Type) {
		return fmt.Errorf("unsupported ruleset rule %q", r.Type)
	}
	p := r.Parameters
	switch r.Type {
	case "pull_request":
		if p.hasChecks() {
			return fmt.Errorf("status-check parameters do not belong to pull_request")
		}
		if complete && (p == nil || p.DismissStaleReviewsOnPush == nil || p.RequireCodeOwnerReview == nil || p.RequireLastPushApproval == nil || p.RequiredApprovingReviewCount == nil || p.RequiredReviewThreadResolution == nil) {
			return fmt.Errorf("pull_request requires complete review parameters")
		}
		if p == nil {
			return nil
		}
		if p.RequiredApprovingReviewCount != nil && *p.RequiredApprovingReviewCount < 0 {
			return fmt.Errorf("required_approving_review_count must be nonnegative")
		}
		if p.AllowedMergeMethods != nil {
			if len(*p.AllowedMergeMethods) == 0 {
				return fmt.Errorf("allowed_merge_methods must be nonempty")
			}
			seen := map[string]bool{}
			for _, v := range *p.AllowedMergeMethods {
				if !slices.Contains([]string{"merge", "squash", "rebase"}, v) || seen[v] {
					return fmt.Errorf("invalid or duplicate merge method %q", v)
				}
				seen[v] = true
			}
		}
		if d := p.DismissalRestriction; d != nil {
			if complete && (d.Enabled == nil || d.AllowedActors == nil) {
				return fmt.Errorf("dismissal_restriction requires enabled and allowed_actors")
			}
			if d.AllowedActors != nil {
				seen := map[string]bool{}
				for _, a := range *d.AllowedActors {
					if a.ID <= 0 || !slices.Contains([]string{"User", "Team", "IntegrationInstallation", "RepositoryRole"}, a.Type) || seen[reviewActorKey(a)] {
						return fmt.Errorf("invalid or duplicate dismissal actor")
					}
					seen[reviewActorKey(a)] = true
				}
			}
		}
		if p.RequiredReviewers != nil {
			if len(*p.RequiredReviewers) > 15 {
				return fmt.Errorf("at most 15 required reviewers are supported")
			}
			seen := map[string]bool{}
			for _, v := range *p.RequiredReviewers {
				key := reviewActorKey(v.Reviewer)
				if v.Reviewer.Type != "Team" || v.Reviewer.ID <= 0 || seen[key] {
					return fmt.Errorf("invalid or duplicate required reviewer")
				}
				seen[key] = true
				if complete && (v.MinimumApprovals == nil || v.FilePatterns == nil) {
					return fmt.Errorf("new required reviewer requires minimum_approvals and file_patterns")
				}
				if v.MinimumApprovals != nil && (*v.MinimumApprovals < 0 || *v.MinimumApprovals > 10) {
					return fmt.Errorf("minimum_approvals must be between 0 and 10")
				}
				if v.FilePatterns != nil {
					for _, pattern := range *v.FilePatterns {
						if !validRulesetText(pattern) {
							return fmt.Errorf("invalid reviewer file pattern")
						}
					}
				}
			}
		}
	case "required_status_checks":
		if p.hasPR() {
			return fmt.Errorf("pull-request parameters do not belong to required_status_checks")
		}
		if complete && (p == nil || p.RequiredStatusChecks == nil || p.StrictRequiredStatusChecksPolicy == nil) {
			return fmt.Errorf("required_status_checks requires checks and strict policy")
		}
		if p == nil {
			return nil
		}
		if p.RequiredStatusChecks != nil {
			seen := map[string]bool{}
			for _, c := range *p.RequiredStatusChecks {
				key := checkKey(c)
				if !validRulesetText(c.Context) || (c.IntegrationID != nil && *c.IntegrationID <= 0) || seen[key] {
					return fmt.Errorf("invalid or duplicate required status check")
				}
				seen[key] = true
			}
		}
	default:
		if p.hasPR() || p.hasChecks() {
			return fmt.Errorf("rule %q has no parameters", r.Type)
		}
	}
	return nil
}
func copyRuleValue[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
func copyRuleList[T any](p *[]T) *[]T {
	if p == nil {
		return nil
	}
	v := append([]T{}, (*p)...)
	return &v
}
func cloneRuleParameters(p *RuleParameters) *RuleParameters {
	if p == nil {
		return nil
	}
	out := *p
	out.AllowedMergeMethods = copyRuleList(p.AllowedMergeMethods)
	if out.AllowedMergeMethods != nil {
		slices.Sort(*out.AllowedMergeMethods)
	}
	out.DismissStaleReviewsOnPush = copyRuleValue(p.DismissStaleReviewsOnPush)
	out.RequireCodeOwnerReview = copyRuleValue(p.RequireCodeOwnerReview)
	out.RequireExtraApprovalForUnattributedChanges = copyRuleValue(p.RequireExtraApprovalForUnattributedChanges)
	out.RequireLastPushApproval = copyRuleValue(p.RequireLastPushApproval)
	out.RequiredApprovingReviewCount = copyRuleValue(p.RequiredApprovingReviewCount)
	out.RequiredReviewThreadResolution = copyRuleValue(p.RequiredReviewThreadResolution)
	if p.DismissalRestriction != nil {
		d := *p.DismissalRestriction
		d.Enabled = copyRuleValue(d.Enabled)
		d.AllowedActors = copyRuleList(d.AllowedActors)
		if d.AllowedActors != nil {
			slices.SortFunc(*d.AllowedActors, compareReviewActor)
		}
		out.DismissalRestriction = &d
	}
	out.RequiredReviewers = copyRuleList(p.RequiredReviewers)
	if out.RequiredReviewers != nil {
		for i := range *out.RequiredReviewers {
			r := &(*out.RequiredReviewers)[i]
			r.MinimumApprovals = copyRuleValue(r.MinimumApprovals)
			r.FilePatterns = copyRuleList(r.FilePatterns)
		}
		slices.SortFunc(*out.RequiredReviewers, func(a, b RequiredReviewer) int { return compareReviewActor(a.Reviewer, b.Reviewer) })
	}
	out.RequiredStatusChecks = copyRuleList(p.RequiredStatusChecks)
	if out.RequiredStatusChecks != nil {
		for i := range *out.RequiredStatusChecks {
			c := &(*out.RequiredStatusChecks)[i]
			c.IntegrationID = copyRuleValue(c.IntegrationID)
		}
		slices.SortFunc(*out.RequiredStatusChecks, func(a, b RequiredStatusCheck) int {
			if v := strings.Compare(a.Context, b.Context); v != 0 {
				return v
			}
			var x, y int64
			if a.IntegrationID != nil {
				x = *a.IntegrationID
			}
			if b.IntegrationID != nil {
				y = *b.IntegrationID
			}
			if x < y {
				return -1
			}
			if x > y {
				return 1
			}
			return 0
		})
	}
	out.StrictRequiredStatusChecksPolicy = copyRuleValue(p.StrictRequiredStatusChecksPolicy)
	out.DoNotEnforceOnCreate = copyRuleValue(p.DoNotEnforceOnCreate)
	return &out
}
func compareReviewActor(a, b ReviewActor) int {
	if v := strings.Compare(a.Type, b.Type); v != 0 {
		return v
	}
	if a.ID < b.ID {
		return -1
	}
	if a.ID > b.ID {
		return 1
	}
	return 0
}
func cloneRulesetRule(r RulesetRule) RulesetRule {
	r.Parameters = cloneRuleParameters(r.Parameters)
	if r.Parameters != nil && !r.Parameters.hasPR() && !r.Parameters.hasChecks() {
		r.Parameters = nil
	}
	return r
}
func mergeRuleParameters(before, desired *RuleParameters) *RuleParameters {
	out := cloneRuleParameters(before)
	d := cloneRuleParameters(desired)
	if d == nil {
		return out
	}
	if out == nil {
		out = &RuleParameters{}
	}
	if d.AllowedMergeMethods != nil {
		out.AllowedMergeMethods = d.AllowedMergeMethods
	}
	if d.DismissStaleReviewsOnPush != nil {
		out.DismissStaleReviewsOnPush = d.DismissStaleReviewsOnPush
	}
	if d.RequireCodeOwnerReview != nil {
		out.RequireCodeOwnerReview = d.RequireCodeOwnerReview
	}
	if d.RequireExtraApprovalForUnattributedChanges != nil {
		out.RequireExtraApprovalForUnattributedChanges = d.RequireExtraApprovalForUnattributedChanges
	}
	if d.RequireLastPushApproval != nil {
		out.RequireLastPushApproval = d.RequireLastPushApproval
	}
	if d.RequiredApprovingReviewCount != nil {
		out.RequiredApprovingReviewCount = d.RequiredApprovingReviewCount
	}
	if d.RequiredReviewThreadResolution != nil {
		out.RequiredReviewThreadResolution = d.RequiredReviewThreadResolution
	}
	if d.DismissalRestriction != nil {
		if out.DismissalRestriction == nil {
			out.DismissalRestriction = &ReviewDismissalRestriction{}
		}
		if d.DismissalRestriction.Enabled != nil {
			out.DismissalRestriction.Enabled = d.DismissalRestriction.Enabled
		}
		if d.DismissalRestriction.AllowedActors != nil {
			out.DismissalRestriction.AllowedActors = d.DismissalRestriction.AllowedActors
		}
	}
	if d.RequiredReviewers != nil {
		old := map[string]RequiredReviewer{}
		if out.RequiredReviewers != nil {
			for _, r := range *out.RequiredReviewers {
				old[reviewActorKey(r.Reviewer)] = r
			}
		}
		for i := range *d.RequiredReviewers {
			r := &(*d.RequiredReviewers)[i]
			if prior, ok := old[reviewActorKey(r.Reviewer)]; ok {
				if r.MinimumApprovals == nil {
					r.MinimumApprovals = prior.MinimumApprovals
				}
				if r.FilePatterns == nil {
					r.FilePatterns = prior.FilePatterns
				}
			}
		}
		out.RequiredReviewers = d.RequiredReviewers
	}
	if d.RequiredStatusChecks != nil {
		out.RequiredStatusChecks = d.RequiredStatusChecks
	}
	if d.StrictRequiredStatusChecksPolicy != nil {
		out.StrictRequiredStatusChecksPolicy = d.StrictRequiredStatusChecksPolicy
	}
	if d.DoNotEnforceOnCreate != nil {
		out.DoNotEnforceOnCreate = d.DoNotEnforceOnCreate
	}
	return cloneRuleParameters(out)
}
