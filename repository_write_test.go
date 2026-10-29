package main

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

type accessPrerequisiteFixtureFailure struct{ error }

func (*accessPrerequisiteFixtureFailure) ReadFailedBeforeWrite() {}
func (failure *accessPrerequisiteFixtureFailure) Unwrap() error  { return failure.error }

func TestRepositoryWriteCompositionPreservesFailureEvidence(t *testing.T) {
	for _, test := range []struct {
		name                                      string
		rest, restFailure, accessFailure, preread bool
	}{
		{"access-only prerequisite failure", false, false, true, true},
		{"mixed prerequisite failure", true, false, true, true},
		{"mixed ordinary access failure", true, false, true, false},
		{"REST failure stops access", true, true, false, false},
		{"mixed success", true, false, false, false},
		{"access-only success", false, false, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			yes, no, policy := true, false, "ALL"
			desired := settings.RepositoryDeclaration{HasPullRequests: &yes, PullRequestCreationPolicy: &policy, IssueCreationPolicy: &policy, HasSponsorships: &no}
			if test.rest {
				desired.HasIssues = &yes
			}
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			cause := errors.New("fixture operation failed")
			var calls []string
			result, err := app.Apply(func() (settings.Declaration, error) { return settings.Declaration{Repository: desired}, nil }, func() (repository.Repository, error) { return target, nil }, app.Writer{Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) error {
				return writeRepositoryWithWriters(repo, d, func(got repository.Repository, rest settings.RepositoryDeclaration) error {
					calls = append(calls, "REST")
					if got != target || !reflect.DeepEqual(rest, settings.RepositoryDeclaration{HasIssues: &yes}) {
						t.Fatal("access intent leaked into REST or target changed", got, rest)
					}
					if test.restFailure {
						return cause
					}
					return nil
				}, func(got repository.Repository, access settings.RepositoryDeclaration) error {
					calls = append(calls, "access")
					if got != target || !reflect.DeepEqual(access, desired) {
						t.Fatal("access declaration/target changed", got, access)
					}
					if test.accessFailure {
						if test.preread {
							return fmt.Errorf("access prerequisite: %w", &accessPrerequisiteFixtureFailure{cause})
						}
						return cause
					}
					return nil
				})
			}})
			wantCalls := []string{"access"}
			if test.rest {
				wantCalls = []string{"REST", "access"}
			}
			if test.restFailure {
				wantCalls = []string{"REST"}
			}
			if !reflect.DeepEqual(calls, wantCalls) {
				t.Fatal("write order/short-circuit changed", calls, wantCalls)
			}
			if !test.restFailure && !test.accessFailure {
				if err != nil || !result.Completed() {
					t.Fatal("successful composition failed", result, err)
				}
				return
			}
			var effects app.RepositoryWriteEffectsFailure
			var preread app.PreWriteReadFailure
			wantEffects := test.rest && test.accessFailure && !test.restFailure
			if !errors.Is(err, cause) || errors.As(err, &effects) != wantEffects || errors.As(err, &preread) != (test.preread && test.accessFailure) {
				t.Fatal("composition lost error evidence", err)
			}
			wantOutcome, wantStatus := app.ApplyFailed, app.OperationFailedUnknown
			if wantEffects {
				wantOutcome = app.ApplyPartialFailure
			} else if test.preread {
				wantStatus = app.OperationFailedRead
			}
			if result.Outcome != wantOutcome || len(result.Operations) != 1 || result.Operations[0].Status != wantStatus {
				t.Fatal("composite effect certainty was misreported", result, err)
			}
		})
	}
}
