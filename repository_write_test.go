package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

type accessPrerequisiteFixtureFailure struct{ error }

func (*accessPrerequisiteFixtureFailure) ReadFailedBeforeWrite() {}
func (failure *accessPrerequisiteFixtureFailure) Unwrap() error  { return failure.error }

func TestRepositoryWriteCompositionPreservesActualProgress(t *testing.T) {
	for _, test := range []struct {
		name        string
		rest        string
		access      string
		wantWrote   bool
		wantEffects bool
		wantOutcome app.ApplyOutcome
		wantStatus  app.ApplyOperationStatus
	}{
		{"matching REST only", "unchanged", "", false, false, app.ApplyMatchingLabels, app.OperationUnchanged},
		{"changed REST only", "changed", "", true, false, app.ApplyCompleted, app.OperationCompleted},
		{"access only", "", "changed", true, false, app.ApplyCompleted, app.OperationCompleted},
		{"access prerequisite failure", "", "preread", false, false, app.ApplyFailed, app.OperationFailedRead},
		{"access uncertain failure", "", "mutation", false, false, app.ApplyFailed, app.OperationFailedUnknown},
		{"matching REST then access", "unchanged", "changed", true, false, app.ApplyCompleted, app.OperationCompleted},
		{"changed REST then access", "changed", "changed", true, false, app.ApplyCompleted, app.OperationCompleted},
		{"matching REST then access prerequisite failure", "unchanged", "preread", false, false, app.ApplyFailed, app.OperationFailedRead},
		{"changed REST then access prerequisite failure", "changed", "preread", true, true, app.ApplyPartialFailure, app.OperationFailedUnknown},
		{"matching REST then access uncertain failure", "unchanged", "mutation", false, false, app.ApplyFailed, app.OperationFailedUnknown},
		{"changed REST then access uncertain failure", "changed", "mutation", true, true, app.ApplyPartialFailure, app.OperationFailedUnknown},
		{"REST prerequisite failure stops access", "preread", "changed", false, false, app.ApplyFailed, app.OperationFailedRead},
		{"REST mutation failure stops access", "mutation", "changed", false, false, app.ApplyFailed, app.OperationFailedUnknown},
		{"REST verification failure stops access", "readback", "changed", true, false, app.ApplyFailed, app.OperationFailedUnknown},
	} {
		t.Run(test.name, func(t *testing.T) {
			yes, no, policy := true, false, "ALL"
			var desired settings.RepositoryDeclaration
			if test.rest != "" {
				desired.HasIssues = &yes
			}
			if test.access != "" {
				desired.HasPullRequests = &yes
				desired.PullRequestCreationPolicy = &policy
				desired.IssueCreationPolicy = &policy
				desired.HasSponsorships = &no
			}
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			cause := errors.New("fixture operation failed")
			var calls []string
			var wrote bool
			result, err := app.Apply(func() (settings.Declaration, error) { return settings.Declaration{Repository: desired}, nil }, func() (repository.Repository, error) { return target, nil }, app.Writer{Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) (bool, error) {
				var err error
				wrote, err = writeRepositoryWithWriters(repo, d, func(got repository.Repository, rest settings.RepositoryDeclaration) (bool, error) {
					calls = append(calls, "REST")
					if got != target || !reflect.DeepEqual(rest, settings.RepositoryDeclaration{HasIssues: &yes}) {
						t.Fatal("access intent leaked into REST or target changed", got, rest)
					}
					switch test.rest {
					case "preread":
						return false, fmt.Errorf("REST prerequisite: %w", &accessPrerequisiteFixtureFailure{cause})
					case "mutation":
						return false, cause
					case "readback":
						return true, cause
					}
					return test.rest == "changed", nil
				}, func(got repository.Repository, access settings.RepositoryDeclaration) error {
					calls = append(calls, "access")
					if got != target || !reflect.DeepEqual(access, desired) {
						t.Fatal("original access declaration or target changed", got, access)
					}
					switch test.access {
					case "preread":
						return fmt.Errorf("access prerequisite: %w", &accessPrerequisiteFixtureFailure{cause})
					case "mutation":
						return cause
					}
					return nil
				})
				return wrote, err
			}})
			restFailed := test.rest == "preread" || test.rest == "mutation" || test.rest == "readback"
			wantCalls := []string{}
			if test.rest != "" {
				wantCalls = append(wantCalls, "REST")
			}
			if test.access != "" && !restFailed {
				wantCalls = append(wantCalls, "access")
			}
			if !reflect.DeepEqual(calls, wantCalls) || wrote != test.wantWrote {
				t.Fatalf("calls=%v wrote=%t, want %v %t", calls, wrote, wantCalls, test.wantWrote)
			}
			if result.Outcome != test.wantOutcome || len(result.Operations) != 1 || result.Operations[0].Kind != app.RepositoryUpdate || result.Operations[0].Status != test.wantStatus {
				t.Fatal("composite progress was misreported", result, err)
			}
			wantFailure := restFailed || test.access == "preread" || test.access == "mutation"
			if !wantFailure {
				if err != nil || !result.Completed() {
					t.Fatal("successful composition failed", result, err)
				}
				return
			}
			var effects app.RepositoryWriteEffectsFailure
			var preread app.PreWriteReadFailure
			wantPreread := test.rest == "preread" || (!restFailed && test.access == "preread")
			if !errors.Is(err, cause) || errors.As(err, &effects) != test.wantEffects || errors.As(err, &preread) != wantPreread {
				t.Fatal("composition lost error evidence", err)
			}
			if strings.Contains(err.Error(), "earlier repository writes remain applied") != test.wantEffects {
				t.Fatal("diagnostic did not follow actual earlier REST progress", err)
			}
		})
	}
}
