package settings

import "testing"

func creationPolicyObservation() RepositoryObservation {
	return NewRepositoryObservation(&RepositoryState{DefaultBranch: "main"}, nil, &PRAccessState{HasPullRequests: true, PullRequestCreationPolicy: "ALL", Normalization: ObservationNormalization{Path: "repository.settings.general.features.pull_request_permissions.creation_allowed_by", RawValue: "FUTURE_PR", Fallback: "all_users"}}).WithIssueCreationPolicy(IssueCreationPolicyState{Policy: "ALL", Normalization: ObservationNormalization{Path: "repository.settings.general.features.issue_permissions.creation_allowed_by", RawValue: "FUTURE_ISSUE", Fallback: "all_users"}})
}

func TestNormalizedCreationComparisonRemainsActionable(t *testing.T) {
	for _, policy := range []string{"ALL", "COLLABORATORS_ONLY"} {
		yes := true
		desired := RepositoryDeclaration{HasIssues: &yes, PullRequestCreationPolicy: &policy, IssueCreationPolicy: &policy}
		result, err := CompareRepositoryObservation(creationPolicyObservation(), desired)
		if err != nil || result.Managed != 3 || len(result.Changes) != 3 || len(result.CommitMessages) != 0 {
			t.Fatal("normalized policies collapsed or duplicated", policy, result, err)
		}
		for i, want := range []string{"repository.has_issues", "repository.pull_request_creation_policy", "repository.issue_creation_policy"} {
			if result.Changes[i].Field().Path() != want {
				t.Fatal("native field order changed", result.Changes)
			}
			if i > 0 {
				change := result.Changes[i]
				if change.Normalization() == (ObservationNormalization{}) {
					t.Fatal("policy provenance lost", change)
				}
				change.Visit(func(bool, bool) { t.Fatal("boolean policy") }, func(before, after string) {
					if before != "ALL" || after != policy {
						t.Fatal("effective pair changed", before, after)
					}
				}, func([]string, []string) { t.Fatal("topics policy") })
			}
		}
	}
	no := false
	observed := creationPolicyObservation()
	observed.issueCreationNormalization.Path = "malformed unmanaged metadata"
	result, err := CompareRepositoryObservation(observed, RepositoryDeclaration{HasIssues: &no})
	if err != nil || result.Managed != 1 || len(result.Changes) != 0 {
		t.Fatal("unmanaged policy was exposed or validated as intent", result, err)
	}
	for _, before := range []string{"ALL", "COLLABORATORS_ONLY"} {
		desired := "ALL"
		observed := NewRepositoryObservation(nil, nil, &PRAccessState{PullRequestCreationPolicy: before})
		result, err := CompareRepositoryObservation(observed, RepositoryDeclaration{PullRequestCreationPolicy: &desired})
		want := 0
		if before != desired {
			want = 1
		}
		if err != nil || result.Managed != 1 || len(result.Changes) != want {
			t.Fatal("canonical comparison contract changed", before, result, err)
		}
		if want == 1 && result.Changes[0].Normalization() != (ObservationNormalization{}) {
			t.Fatal("canonical policy gained provenance")
		}
	}
}

func TestNormalizedCreationComparisonRejectsPartialOrForgedEvidence(t *testing.T) {
	for _, corrupt := range []func(*RepositoryObservation, *RepositoryDeclaration){
		func(o *RepositoryObservation, _ *RepositoryDeclaration) { o.prCreationNormalization.Path = "wrong" },
		func(o *RepositoryObservation, _ *RepositoryDeclaration) {
			o.prCreationNormalization.Fallback = "collaborators_only"
		},
		func(o *RepositoryObservation, _ *RepositoryDeclaration) {
			o.prCreationNormalization.RawValue = "COLLABORATORS_ONLY"
		},
		func(o *RepositoryObservation, _ *RepositoryDeclaration) {
			value := "COLLABORATORS_ONLY"
			o.values.PullRequestCreationPolicy = &value
		},
		func(o *RepositoryObservation, _ *RepositoryDeclaration) { o.values.PullRequestCreationPolicy = nil },
		func(_ *RepositoryObservation, d *RepositoryDeclaration) {
			value := "FUTURE"
			d.PullRequestCreationPolicy = &value
		},
		func(o *RepositoryObservation, _ *RepositoryDeclaration) { o.issueCreationNormalization.Path = "wrong" },
	} {
		yes, policy := true, "ALL"
		observed := creationPolicyObservation()
		desired := RepositoryDeclaration{HasIssues: &yes, PullRequestCreationPolicy: &policy, IssueCreationPolicy: &policy}
		corrupt(&observed, &desired)
		result, err := CompareRepositoryObservation(observed, desired)
		if err == nil || result.Managed != 0 || result.HasChanges() {
			t.Fatal("forged evidence exposed partial comparison", result, err)
		}
	}
}
