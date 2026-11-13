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

func TestClassicPrunePreflightBeforeAnyWrite(t *testing.T) {
	for _, mode := range []string{"incomplete", "collision", "missing-id", "archived"} {
		t.Run(mode, func(t *testing.T) {
			observed := appClassicObserved()
			rules := []settings.ClassicRule{}
			description := "changed"
			no := false
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description, Archived: &no}, BranchProtectionRules: &rules}
			switch mode {
			case "incomplete":
				observed.Rules[0].Companions.Checks.Present = false
			case "collision":
				rules = append(rules, settings.ClassicRule{BranchNamePattern: "main"})
			case "missing-id":
				id := int64(9)
				rules = append(rules, settings.ClassicRule{ID: &id, BranchNamePattern: "missing"})
			case "archived":
				observed.Archived = true
				d.Repository.Archived = nil
			}
			prune, _ := settings.ParsePruneScopes("branch-protection-rules")
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
				ReadBranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) { return observed, nil },
				DeleteBranchProtectionRule: func(repository.Repository, settings.ClassicPruneCandidate) error {
					t.Fatal("deleted before preflight")
					return nil
				},
				Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
					t.Fatal("wrote before preflight")
					return false, nil
				},
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					t.Fatal("passed global preflight")
					return settings.ArchiveState{}, nil
				},
			}, prune)
			if err == nil || result.Outcome != ApplyFailed || len(result.Operations) != 0 || !strings.Contains(err.Error(), "before any writes") {
				t.Fatal(result, err)
			}
		})
	}
}

func TestClassicPruneFixedCandidatesOrderingAndProgress(t *testing.T) {
	for _, mode := range []string{"success", "writeback-failure", "candidate-drift", "late-arrival", "empty-selection", "second-delete-read", "second-delete-unknown", "noop-before-read", "write-before-read", "own-unarchive"} {
		t.Run(mode, func(t *testing.T) {
			observed := appClassicObserved()
			second := appClassicObserved().Rules[0]
			id := int64(8)
			second.Rule.ID = &id
			second.NodeID = "B8"
			second.Rule.BranchNamePattern = "old/**"
			observed.Rules = append(observed.Rules, second)
			rules := []settings.ClassicRule{{BranchNamePattern: "new/**"}}
			if mode == "noop-before-read" || mode == "write-before-read" || mode == "own-unarchive" || mode == "second-delete-read" || mode == "second-delete-unknown" {
				rules = []settings.ClassicRule{}
			}
			if mode == "empty-selection" {
				observed.Rules = []settings.ClassicState{}
				rules = []settings.ClassicRule{}
			}
			d := settings.Declaration{BranchProtectionRules: &rules}
			if mode == "noop-before-read" || mode == "write-before-read" {
				description := "same"
				d.Repository.Description = &description
			}
			if mode == "own-unarchive" {
				observed.Archived = true
				no := false
				d.Repository.Archived = &no
			}
			events := []string{}
			reads := 0
			deletes := 0
			prune, _ := settings.ParsePruneScopes("all")
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
				ReadBranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) {
					events = append(events, "read")
					reads++
					if reads == 2 {
						switch mode {
						case "candidate-drift":
							*observed.Rules[0].Rule.DoNotAllowBypassingTheAboveSettings = true
						case "late-arrival", "empty-selection":
							late := appClassicObserved().Rules[0]
							n := int64(9)
							late.Rule.ID = &n
							late.NodeID = "B9"
							late.Rule.BranchNamePattern = "late/**"
							observed.Rules = append(observed.Rules, late)
						case "own-unarchive":
							observed.Archived = false
						}
					}
					return observed, nil
				},
				Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
					events = append(events, "repository")
					return mode == "write-before-read", nil
				},
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					events = append(events, "archive-read")
					return settings.ArchiveState{ID: 1, FullName: "daiksud/gh-settings", Archived: true}, nil
				},
				Archive: func(repository.Repository, settings.ArchiveState, bool) error {
					events = append(events, "unarchive")
					return nil
				},
				BranchProtectionRule: func(_ repository.Repository, s settings.ClassicSelection) (settings.ClassicWriteResult, error) {
					events = append(events, "create")
					return settings.ClassicWriteResult{ID: 10, NodeID: "B10", Wrote: true}, nil
				},
				CreatedID: func(string, string, int64) error {
					events = append(events, "writeback")
					if mode == "writeback-failure" {
						return errors.New("editor changed file")
					}
					return nil
				},
				DeleteBranchProtectionRule: func(_ repository.Repository, c settings.ClassicPruneCandidate) error {
					events = append(events, fmt.Sprintf("delete:%d", c.ID()))
					deletes++
					if (deletes == 2 && mode == "second-delete-read") || mode == "noop-before-read" || mode == "write-before-read" {
						return &environmentTestReadFailure{errors.New("drift")}
					}
					if deletes == 2 && mode == "second-delete-unknown" {
						return errors.New("unconfirmed ACK")
					}
					if c.ID() != 7 && c.ID() != 8 {
						t.Fatal("new arrival adopted", c.ID())
					}
					return nil
				},
			}, prune)
			bad := strings.Contains(mode, "failure") || strings.Contains(mode, "drift") || strings.Contains(mode, "-read") || mode == "second-delete-unknown"
			if (err != nil) != bad {
				t.Fatal(result, events, err)
			}
			if mode == "success" || mode == "late-arrival" {
				if !reflect.DeepEqual(events, []string{"read", "read", "create", "writeback", "delete:7", "delete:8"}) {
					t.Fatal(events)
				}
			}
			if mode == "writeback-failure" && (deletes != 0 || result.Outcome != ApplyPartialFailure) {
				t.Fatal(result, events, err)
			}
			if mode == "candidate-drift" && (!reflect.DeepEqual(events, []string{"read", "read"}) || result.Outcome != ApplyFailed) {
				t.Fatal(result, events, err)
			}
			if mode == "empty-selection" && (deletes != 0 || len(result.Operations) != 1) {
				t.Fatal("late candidate was added", result, events)
			}
			if mode == "second-delete-read" || mode == "second-delete-unknown" {
				want := OperationFailedRead
				if mode == "second-delete-unknown" {
					want = OperationFailedUnknown
				}
				if result.Outcome != ApplyPartialFailure || result.Operations[len(result.Operations)-1].Status != want || deletes != 2 {
					t.Fatal(result, events, err)
				}
			}
			if mode == "noop-before-read" || mode == "write-before-read" {
				if (result.Outcome == ApplyPartialFailure) != (mode == "write-before-read") || deletes != 1 || result.Operations[len(result.Operations)-1].Status != OperationNotExecuted {
					t.Fatal(result, events, err)
				}
			}
			if mode == "own-unarchive" && !reflect.DeepEqual(events, []string{"read", "archive-read", "unarchive", "read", "delete:7", "delete:8"}) {
				t.Fatal(events)
			}
		})
	}
}

func TestClassicPruneOmissionNeverAcquires(t *testing.T) {
	prune, _ := settings.ParsePruneScopes("all")
	result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{}, nil }, checkoutRepository, Writer{ReadBranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) {
		t.Fatal("omitted collection read")
		return settings.ClassicObservation{}, nil
	}}, prune)
	if err != nil || result.Outcome != ApplyUnmanaged {
		t.Fatal(result, err)
	}
}
