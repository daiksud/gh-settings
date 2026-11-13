package app

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestMatchingRepositoryApplyRecordsNoWrite(t *testing.T) {
	description := "unchanged"
	desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}}
	calls := 0
	result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, checkoutRepository, Writer{
		Repository: func(_ repository.Repository, got settings.RepositoryDeclaration) (bool, error) {
			calls++
			if !reflect.DeepEqual(got, desired.Repository) {
				t.Fatal("declaration changed before reconciliation", got)
			}
			return false, nil
		},
	})
	if err != nil || calls != 1 || !result.Completed() || result.Outcome != ApplyMatchingLabels || !reflect.DeepEqual(result.Operations, []ApplyOperation{{Kind: RepositoryUpdate, Status: OperationUnchanged}}) {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls)
	}
}

func TestLabelAcquisitionFailureReportsActualEarlierRepositoryWrites(t *testing.T) {
	for _, wrote := range []bool{false, true} {
		for _, failure := range []string{"read", "comparison"} {
			t.Run(fmt.Sprintf("wrote=%t/%s", wrote, failure), func(t *testing.T) {
				description, color, id := "declared", "abcdef", int64(7)
				labels := appLabelDeclarations(t, settings.LabelDeclaration{ID: &id, Name: "bug", Color: &color})
				desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}, Labels: &labels}
				cause := errors.New("label read unavailable")
				var calls []string
				result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, checkoutRepository, Writer{
					Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
						calls = append(calls, "repository")
						return wrote, nil
					},
					ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
						calls = append(calls, "labels")
						if failure == "read" {
							return settings.LabelObservation{}, cause
						}
						// A complete empty inventory cannot satisfy the selected ID.
						return settings.NewLabelObservation(nil)
					},
					UpdateLabel: func(repository.Repository, settings.LabelChange) error {
						t.Fatal("failed acquisition attempted a label write")
						return nil
					},
				})
				wantOutcome, repositoryStatus := ApplyFailed, OperationUnchanged
				if wrote {
					wantOutcome, repositoryStatus = ApplyPartialFailure, OperationCompleted
				}
				wantOperations := []ApplyOperation{{Kind: RepositoryUpdate, Status: repositoryStatus}, {Kind: LabelsRead, Status: OperationFailedRead}}
				if err == nil || result.Outcome != wantOutcome || !reflect.DeepEqual(result.Operations, wantOperations) || !reflect.DeepEqual(calls, []string{"repository", "labels"}) {
					t.Fatalf("result=%+v error=%v calls=%v", result, err, calls)
				}
				if failure == "read" && !errors.Is(err, cause) {
					t.Fatal("lost original read failure", err)
				}
				if strings.Contains(err.Error(), "earlier resource writes remain applied") != wrote {
					t.Fatal("diagnostic did not follow actual earlier writes", err)
				}
			})
		}
	}
}

func TestArchiveOrderingWithMatchingRepositorySettings(t *testing.T) {
	for _, test := range []struct {
		name         string
		initial      bool
		archive      bool
		failure      string
		wantCalls    []string
		wantOutcome  ApplyOutcome
		wantStatuses []ApplyOperationStatus
	}{
		{"already active", false, false, "", []string{"read", "repository"}, ApplyMatchingLabels, []ApplyOperationStatus{OperationCompleted, OperationUnchanged}},
		{"unarchive first", true, false, "", []string{"read", "unarchive", "repository"}, ApplyCompleted, []ApplyOperationStatus{OperationCompleted, OperationCompleted, OperationUnchanged}},
		{"archive last", false, true, "", []string{"read", "repository", "read", "archive"}, ApplyCompleted, []ApplyOperationStatus{OperationCompleted, OperationUnchanged, OperationCompleted, OperationCompleted}},
		{"final archive read fails", false, true, "read", []string{"read", "repository", "read"}, ApplyFailed, []ApplyOperationStatus{OperationCompleted, OperationUnchanged, OperationFailedRead}},
		{"final archive mutation fails", false, true, "mutation", []string{"read", "repository", "read", "archive"}, ApplyFailed, []ApplyOperationStatus{OperationCompleted, OperationUnchanged, OperationCompleted, OperationFailedUnknown}},
	} {
		t.Run(test.name, func(t *testing.T) {
			description := "unchanged"
			desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description, Archived: &test.archive}}
			cause := errors.New("archive operation failed")
			var calls []string
			reads := 0
			result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, checkoutRepository, Writer{
				ReadArchive: func(repo repository.Repository) (settings.ArchiveState, error) {
					calls = append(calls, "read")
					reads++
					if reads == 2 && test.failure == "read" {
						return settings.ArchiveState{}, cause
					}
					return settings.ArchiveState{ID: 1, FullName: repo.Owner + "/" + repo.Name, Archived: test.initial}, nil
				},
				Archive: func(_ repository.Repository, _ settings.ArchiveState, archived bool) error {
					if archived {
						calls = append(calls, "archive")
					} else {
						calls = append(calls, "unarchive")
					}
					if test.failure == "mutation" {
						return cause
					}
					return nil
				},
				Repository: func(_ repository.Repository, got settings.RepositoryDeclaration) (bool, error) {
					calls = append(calls, "repository")
					if got.Archived != nil || got.Description == nil || *got.Description != description {
						t.Fatal("archive intent leaked into ordinary reconciliation", got)
					}
					return false, nil
				},
			})
			if result.Outcome != test.wantOutcome || !reflect.DeepEqual(calls, test.wantCalls) || (err != nil) != (test.failure != "") {
				t.Fatalf("result=%+v error=%v calls=%v", result, err, calls)
			}
			if err != nil && !errors.Is(err, cause) {
				t.Fatal("lost archive failure", err)
			}
			var statuses []ApplyOperationStatus
			for _, operation := range result.Operations {
				statuses = append(statuses, operation.Status)
			}
			if !reflect.DeepEqual(statuses, test.wantStatuses) {
				t.Fatal("matching repository work changed archive progress", result.Operations)
			}
		})
	}
}
