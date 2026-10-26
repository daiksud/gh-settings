package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivePlanIsSelectiveAndNative(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, tt := range []struct {
		from, to bool
		changed  bool
	}{{false, true, true}, {true, false, true}, {true, true, false}, {false, false, false}} {
		calls := 0
		got, err := Plan(func() (settings.Declaration, error) {
			return settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &tt.to}}, nil
		}, func() (repository.Repository, error) { return repo, nil }, Reader{
			Archive: func(got repository.Repository) (settings.ArchiveState, error) {
				calls++
				if got != repo {
					t.Fatal(got)
				}
				return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: tt.from}, nil
			},
			ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
				t.Fatal("unrelated repository read")
				return settings.RepositoryObservation{}, nil
			},
		})
		if err != nil || calls != 1 || got.Comparison.Managed != 1 || (len(got.Comparison.Changes) == 1) != tt.changed {
			t.Fatal(tt, got, err, calls)
		}
		if tt.changed {
			var before, after bool
			got.Comparison.Changes[0].Visit(func(b, a bool) { before, after = b, a }, func(string, string) { t.Fatal("string") }, func([]string, []string) { t.Fatal("topics") })
			if before != tt.from || after != tt.to {
				t.Fatal(before, after)
			}
		}
	}
	got, err := Plan(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: new(bool)}}, nil
	}, func() (repository.Repository, error) { return repo, nil }, Reader{Archive: func(repository.Repository) (settings.ArchiveState, error) { return settings.ArchiveState{}, nil }})
	if err == nil || got.Comparison.Managed != 0 {
		t.Fatal("missing state became false", got, err)
	}
}

func TestArchiveApplyTransitionTableAndOrder(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, tt := range []struct {
		name            string
		from, to, other bool
		want            []string
		reject          bool
		written         bool
	}{
		{"active archive-only", false, true, false, []string{"read", "read", "archive:true"}, false, true},
		{"active archive-after-other", false, true, true, []string{"read", "other", "read", "archive:true"}, false, true},
		{"archived unarchive-only", true, false, false, []string{"read", "archive:false"}, false, true},
		{"archived unarchive-before-other", true, false, true, []string{"read", "archive:false", "other"}, false, true},
		{"active matching active", false, false, false, []string{"read"}, false, false},
		{"archived matching archived", true, true, false, []string{"read"}, false, false},
		{"archived other forbidden", true, true, true, []string{"read"}, true, false},
		{"active false other", false, false, true, []string{"read", "other"}, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := []string{}
			archived := tt.to
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}}
			if tt.other {
				s := "Example"
				d.Repository.Description = &s
			}
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					calls = append(calls, "read")
					return settings.ArchiveState{ID: 42, FullName: "owner/current", Archived: tt.from}, nil
				},
				Archive: func(_ repository.Repository, initial settings.ArchiveState, desired bool) error {
					calls = append(calls, "archive:"+map[bool]string{false: "false", true: "true"}[desired])
					if initial.ID != 42 {
						t.Fatal(initial)
					}
					return nil
				},
				Repository: func(_ repository.Repository, got settings.RepositoryDeclaration) error {
					calls = append(calls, "other")
					if got.Archived != nil || got.Description == nil {
						t.Fatal(got)
					}
					return nil
				},
			}
			got, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, writer)
			if (err != nil) != tt.reject || !reflect.DeepEqual(calls, tt.want) || (got.Completed() == tt.reject) {
				t.Fatal(tt, calls, got, err)
			}
			if !tt.reject && tt.written && got.Outcome != ApplyCompleted {
				t.Fatal(got)
			}
			if !tt.reject && !tt.written && got.Outcome != ApplyMatchingLabels {
				t.Fatal(got)
			}
		})
	}
}

func TestArchiveApplyFailuresPreserveProvenProgress(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	fail := errors.New("fixture failure")
	for _, tt := range []struct {
		name         string
		initial      bool
		where        string
		want         ApplyOutcome
		failedKind   ApplyOperationKind
		failedStatus ApplyOperationStatus
	}{
		{"initial read", false, "initial", ApplyFailed, ArchiveRead, OperationFailedRead},
		{"unarchive mutation", true, "transition", ApplyFailed, ArchiveUnarchive, OperationFailedUnknown},
		{"normal after unarchive", true, "other", ApplyPartialFailure, RepositoryUpdate, OperationFailedUnknown},
		{"normal before archive", false, "other", ApplyFailed, RepositoryUpdate, OperationFailedUnknown},
		{"final read", false, "final", ApplyPartialFailure, ArchiveRead, OperationFailedRead},
		{"final mutation", false, "transition", ApplyPartialFailure, ArchiveUpdate, OperationFailedUnknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			desired := !tt.initial
			description := "change"
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &desired, Description: &description}}
			reads, writes := 0, 0
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					reads++
					if tt.where == "initial" || (tt.where == "final" && reads == 2) {
						return settings.ArchiveState{}, fail
					}
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: tt.initial}, nil
				},
				Archive: func(repository.Repository, settings.ArchiveState, bool) error {
					writes++
					if tt.where == "transition" {
						return fail
					}
					return nil
				},
				Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
					writes++
					if tt.where == "other" {
						return fail
					}
					return nil
				},
			}
			got, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, writer)
			if err == nil || got.Outcome != tt.want {
				t.Fatal(tt, got, err)
			}
			last := got.Operations[len(got.Operations)-1]
			if last.Kind != tt.failedKind || last.Status != tt.failedStatus {
				t.Fatal(tt, got.Operations)
			}
			if tt.where == "initial" && writes != 0 {
				t.Fatal("read failure wrote", writes)
			}
		})
	}
	// A changed target between ordinary writes and final archive is a failed read, not another PATCH.
	yes := true
	s := "updated"
	reads, writes := 0, 0
	got, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes, Description: &s}}, nil
	}, func() (repository.Repository, error) { return repo, nil }, Writer{
		ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
			reads++
			id := int64(1)
			if reads == 2 {
				id = 2
			}
			return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: false}, nil
		},
		Repository: func(repository.Repository, settings.RepositoryDeclaration) error { return nil },
		Archive:    func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
	})
	if err == nil || got.Outcome != ApplyPartialFailure || writes != 0 || !strings.Contains(err.Error(), "changed") {
		t.Fatal(got, err, writes)
	}
}

func TestArchivedRepositoryWithEmptyLabelSelectionIsMatching(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	archived := true
	labels := appLabelDeclarations(t)
	desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, Labels: &labels}
	reads := 0
	result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, Writer{
		ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
			reads++
			return settings.ArchiveState{ID: 42, FullName: "owner/current", Archived: true}, nil
		},
		ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
			t.Fatal("empty selection read labels")
			return settings.LabelObservation{}, nil
		},
		Archive: func(repository.Repository, settings.ArchiveState, bool) error {
			t.Fatal("matching archive state mutated")
			return nil
		},
	})
	if err != nil || result.Outcome != ApplyMatchingLabels || reads != 1 || len(result.Operations) != 1 {
		t.Fatalf("result=%+v error=%v reads=%d", result, err, reads)
	}
}

func TestArchiveExportExplicitlyOptInAndFailClosed(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	from := settings.RepositoryState{DefaultBranch: "main"}
	merge := settings.MergeMethodsState{SquashMergeCommitTitle: "PR_TITLE", SquashMergeCommitMessage: "PR_BODY", MergeCommitTitle: "PR_TITLE", MergeCommitMessage: "PR_BODY"}
	pr := settings.PRAccessState{PullRequestCreationPolicy: "ALL"}
	full := settings.NewRepositoryObservation(&from, &merge, &pr).WithIssueCreationPolicy(settings.IssueCreationPolicyState{Policy: "ALL"}).WithSponsorships(settings.SponsorshipsState{})
	archiveCalls, saveCalls := 0, 0
	failArchive := false
	reader := Reader{
		ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
			return full, nil
		},
		Topics: func(repository.Repository) ([]string, error) { return []string{}, nil },
		Labels: func(repository.Repository) (settings.LabelObservation, error) {
			return settings.NewLabelObservation([]settings.LabelState{})
		},
		Archive: func(repository.Repository) (settings.ArchiveState, error) {
			archiveCalls++
			if failArchive {
				return settings.ArchiveState{}, errors.New("archive status unavailable")
			}
			return settings.ArchiveState{ID: 42, FullName: "owner/current", Archived: false}, nil
		},
	}
	checkout := ExportCheckout{Repository: func() (repository.Repository, error) { return repo, nil }, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(d settings.Declaration) error {
			saveCalls++
			if d.Repository.Archived != nil && *d.Repository.Archived {
				t.Fatal("false was not preserved")
			}
			return nil
		}, nil
	}}
	if err := Export(checkout, reader); err != nil || archiveCalls != 0 || saveCalls != 1 {
		t.Fatal(err, archiveCalls, saveCalls)
	}
	checkout.IncludeArchive = true
	if err := Export(checkout, reader); err != nil || archiveCalls != 1 || saveCalls != 2 {
		t.Fatal(err, archiveCalls, saveCalls)
	}
	failArchive = true
	if err := Export(checkout, reader); err == nil || archiveCalls != 2 || saveCalls != 2 {
		t.Fatal("failed read mutated export", err, archiveCalls, saveCalls)
	}
}

func TestArchivePlanEvaluatesArchivedMixedDeclaration(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	archived := true
	current, other := "current", "other"
	for _, tt := range []struct {
		name        string
		desired     settings.Declaration
		wantRead    string
		wantChanges int
	}{
		{name: "matching repository", desired: settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived, Description: &current}}, wantRead: "repository"},
		{name: "different repository", desired: settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived, Description: &other}}, wantRead: "repository", wantChanges: 1},
		{name: "matching topics", desired: settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, Topics: settings.TopicsDeclaration{Names: &[]string{"current"}}}, wantRead: "topics"},
		{name: "different topics", desired: settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, Topics: settings.TopicsDeclaration{Names: &[]string{"other"}}}, wantRead: "topics", wantChanges: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := []string{}
			result, err := Plan(func() (settings.Declaration, error) { return tt.desired, nil }, func() (repository.Repository, error) { return repo, nil }, Reader{
				Archive: func(repository.Repository) (settings.ArchiveState, error) {
					calls = append(calls, "archive")
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
					calls = append(calls, "repository")
					return settings.NewRepositoryObservation(&settings.RepositoryState{Description: "current"}, nil, nil), nil
				},
				Topics: func(repository.Repository) ([]string, error) {
					calls = append(calls, "topics")
					return []string{"current"}, nil
				},
			})
			if err != nil || result.Comparison.Managed != 2 || len(result.Comparison.Changes) != tt.wantChanges || !reflect.DeepEqual(calls, []string{"archive", tt.wantRead}) {
				t.Fatal(result, err, calls)
			}
		})
	}
}

func TestArchivePlanFailsClosedOnMixedObservationFailure(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	archived := true
	description := "current"
	result, err := Plan(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived, Description: &description}}, nil
	}, func() (repository.Repository, error) { return repo, nil }, Reader{
		Archive: func(repository.Repository) (settings.ArchiveState, error) {
			return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
		},
		ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
			return settings.RepositoryObservation{}, errors.New("fixture incomplete observation")
		},
	})
	var planErr *PlanError
	if !errors.As(err, &planErr) || planErr.Stage != "read-failure" || !strings.Contains(err.Error(), "incomplete observation") || result.Comparison.Managed != 0 {
		t.Fatal(result, err)
	}
}

func TestArchivePlanTransitionsRespectApplyOrder(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	desc := "after"
	for _, tc := range []struct {
		from, to  bool
		wantFirst string
	}{
		{true, false, "repository.archived"},
		{false, true, "repository.description"},
	} {
		d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &tc.to, Description: &desc}}
		result, err := Plan(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, Reader{
			Archive: func(repository.Repository) (settings.ArchiveState, error) {
				return settings.ArchiveState{ID: 7, FullName: "owner/current", Archived: tc.from}, nil
			},
			ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
				return settings.NewRepositoryObservation(&settings.RepositoryState{Description: "before"}, nil, nil), nil
			},
		})
		if err != nil || len(result.Comparison.Changes) != 2 || result.Comparison.Changes[0].Field().Path() != tc.wantFirst {
			t.Fatal(tc, result, err)
		}
	}
}

// CLI error presentation discards ApplyResult. The diagnostic must therefore
// preserve confirmed unarchive progress even when a later write fails.
func TestArchiveUnarchiveProgressSurvivesSubsequentFailures(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, variant := range []string{"repository", "topics"} {
		active := false
		desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &active}}
		if variant == "repository" {
			value := "new"
			desired.Repository.Description = &value
		}
		if variant == "topics" {
			values := []string{"new"}
			desired.Topics.Names = &values
		}
		result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, Writer{
			ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
				return settings.ArchiveState{ID: 2, FullName: "owner/current", Archived: true}, nil
			},
			Archive: func(repository.Repository, settings.ArchiveState, bool) error { return nil },
			Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
				return errors.New("repository write denied")
			},
			Topics: func(repository.Repository, settings.TopicsDeclaration) error {
				return errors.New("topics write denied")
			},
		})
		if err == nil || result.Outcome != ApplyPartialFailure || !strings.Contains(err.Error(), "repository was unarchived") || !strings.Contains(err.Error(), "earlier changes remain applied") || !strings.Contains(err.Error(), variant+" write denied") {
			t.Fatal(variant, result, err)
		}
		if len(result.Operations) != 3 || result.Operations[1].Kind != ArchiveUnarchive || result.Operations[1].Status != OperationCompleted || result.Operations[2].Status != OperationFailedUnknown {
			t.Fatal(variant, result.Operations)
		}
	}
}
