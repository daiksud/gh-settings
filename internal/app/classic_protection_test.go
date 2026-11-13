package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func appClassicObserved() settings.ClassicObservation {
	id, no, n := int64(7), false, 2
	empty := []string{}
	checks := []settings.ClassicCheck{}
	return settings.ClassicObservation{RepositoryID: "R", Rules: []settings.ClassicState{{NodeID: "B7", RepositoryID: "R", Rule: settings.ClassicRule{ID: &id, BranchNamePattern: "main", AllowDeletions: &no, RequireSignedCommits: &no, RequireLinearHistory: &no, RequireConversationResolutionBeforeMerging: &no, LockBranch: &no, DoNotAllowBypassingTheAboveSettings: &no, RequireAPullRequestBeforeMerging: &settings.ClassicPullRequestReviews{Enabled: &no}, RequireStatusChecksToPassBeforeMerging: &settings.ClassicStatusChecks{Enabled: &no}, RequireDeploymentsToSucceedBeforeMerging: &settings.ClassicDeployments{Enabled: &no}}, Companions: settings.ClassicCompanions{ReviewCount: settings.ClassicNullable[int]{Present: true, Value: &n}, Deployments: settings.ClassicNullable[[]string]{Present: true, Value: &empty}, Contexts: settings.ClassicNullable[[]string]{Present: true, Value: &empty}, Checks: settings.ClassicNullable[[]settings.ClassicCheck]{Present: true, Value: &checks}, PushActors: []settings.ClassicActor{}, ReviewDismissalActors: []settings.ClassicActor{}, ForcePushActors: []settings.ClassicActor{}, PullRequestActors: []settings.ClassicActor{}}}}}
}
func TestClassicApplyOrderIdentityWritebackAndPartialProgress(t *testing.T) {
	for _, mode := range []string{"success", "second-read-failure", "second-write-failure", "writeback-failure", "preflight-collision", "unarchive", "existing-noop"} {
		t.Run(mode, func(t *testing.T) {
			observed := appClassicObserved()
			rules := []settings.ClassicRule{{BranchNamePattern: "z/**"}, {BranchNamePattern: "z/*"}}
			d := settings.Declaration{BranchProtectionRules: &rules}
			events := []string{}
			recorded := []int64{}
			if mode == "preflight-collision" {
				rules[1].BranchNamePattern = "main"
			}
			if mode == "unarchive" {
				no := false
				d.Repository.Archived = &no
			}
			if mode == "existing-noop" {
				rules = []settings.ClassicRule{{ID: observed.Rules[0].Rule.ID, BranchNamePattern: "main"}}
			}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					events = append(events, "archive-read")
					return settings.ArchiveState{ID: 1, FullName: "daiksud/gh-settings", Archived: true}, nil
				}, Archive: func(repository.Repository, settings.ArchiveState, bool) error {
					events = append(events, "unarchive")
					return nil
				},
				ReadBranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) {
					events = append(events, "read")
					return observed, nil
				},
				BranchProtectionRule: func(_ repository.Repository, s settings.ClassicSelection) (settings.ClassicWriteResult, error) {
					events = append(events, s.Desired.BranchNamePattern)
					if s.Before != nil {
						return settings.ClassicWriteResult{ID: 7, NodeID: "B7"}, nil
					}
					id := int64(8)
					if s.Desired.BranchNamePattern == "z/*" {
						id = 9
						if mode == "second-read-failure" {
							return settings.ClassicWriteResult{}, &environmentTestReadFailure{errors.New("read failed")}
						}
						if mode == "second-write-failure" {
							return settings.ClassicWriteResult{ID: 9, NodeID: "B9", Wrote: true}, errors.New("readback failed")
						}
					}
					return settings.ClassicWriteResult{ID: id, NodeID: fmt.Sprint(id), Wrote: true}, nil
				},
				CreatedID: func(collection, name string, id int64) error {
					if collection != "branch_protection_rules" {
						t.Fatal(collection)
					}
					events = append(events, "record "+name)
					if mode == "writeback-failure" {
						return errors.New("concurrent editor")
					}
					recorded = append(recorded, id)
					return nil
				},
			})
			bad := strings.Contains(mode, "failure") || mode == "preflight-collision"
			if (err != nil) != bad {
				t.Fatal(mode, result, events, err)
			}
			if mode == "success" && !reflect.DeepEqual(events, []string{"read", "z/**", "record z/**", "z/*", "record z/*"}) {
				t.Fatal(events)
			}
			if mode == "preflight-collision" && !reflect.DeepEqual(events, []string{"read"}) {
				t.Fatal("partial preflight", events)
			}
			if strings.HasPrefix(mode, "second-") && (result.Outcome != ApplyPartialFailure || !reflect.DeepEqual(recorded, []int64{8})) {
				t.Fatal(result, recorded, err)
			}
			if mode == "existing-noop" && (len(recorded) != 0 || result.Operations[1].Status != OperationUnchanged) {
				t.Fatal(result)
			}
			if mode == "unarchive" && !strings.HasPrefix(strings.Join(events, ","), "archive-read,unarchive,read,") {
				t.Fatal(events)
			}
		})
	}
}
func TestClassicFailureAfterRepositoryNoopIsNotPartial(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		rules := []settings.ClassicRule{{BranchNamePattern: "new"}}
		description := "desired"
		d := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}, BranchProtectionRules: &rules}
		result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { return earlier, nil }, ReadBranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) {
			return settings.ClassicObservation{}, errors.New("denied")
		}})
		if err == nil || (result.Outcome == ApplyPartialFailure) != earlier || strings.Contains(err.Error(), "earlier successful writes") != earlier {
			t.Fatal(result, err)
		}
	}
}
func TestClassicExportRetainsPriorOnUnavailableAndProjectsOnlySupported(t *testing.T) {
	prior := []settings.ClassicRule{{BranchNamePattern: "retained"}}
	for _, denied := range []bool{false, true} {
		var saved settings.Declaration
		info := []string{}
		err := Export(ExportCheckout{Existing: settings.Declaration{BranchProtectionRules: &prior}, Repository: checkoutRepository, Check: github.CheckExportAcquisition, Info: func(path string, _ error) error { info = append(info, path); return nil }, PrepareSave: func() (func(settings.Declaration) error, error) {
			return func(d settings.Declaration) error { saved = d; return nil }, nil
		}}, Reader{BranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) {
			if denied {
				return settings.ClassicObservation{}, errors.New("denied")
			}
			return appClassicObserved(), nil
		}})
		if err != nil || saved.BranchProtectionRules == nil {
			t.Fatal(err)
		}
		if denied {
			if !reflect.DeepEqual(*saved.BranchProtectionRules, prior) {
				t.Fatal(saved)
			}
		} else if *(*saved.BranchProtectionRules)[0].ID != 7 || (*saved.BranchProtectionRules)[0].AllowDeletions == nil {
			t.Fatal(saved)
		}
	}
}

func TestClassicPartialAcknowledgmentRemainsUnknownAndStopsWriteback(t *testing.T) {
	rules := []settings.ClassicRule{{BranchNamePattern: "new/**"}, {BranchNamePattern: "new/*"}}
	calls := 0
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{BranchProtectionRules: &rules}, nil
	}, checkoutRepository, Writer{
		ReadBranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) { return appClassicObserved(), nil },
		BranchProtectionRule: func(repository.Repository, settings.ClassicSelection) (settings.ClassicWriteResult, error) {
			calls++
			return settings.ClassicWriteResult{ID: 101, NodeID: "B_101"}, errors.New("partial GraphQL acknowledgment")
		},
		CreatedID: func(string, string, int64) error { t.Fatal("unverified ID was written back"); return nil },
	})
	if err == nil || calls != 1 || result.Outcome != ApplyFailed || len(result.Operations) != 3 || result.Operations[1].Status != OperationFailedUnknown || result.Operations[2].Status != OperationNotExecuted {
		t.Fatal(result, calls, err)
	}
	if !reflect.DeepEqual(result.CreatedIDs, []CreatedObject{{Collection: "branch_protection_rules", Name: "new/**", ID: 101}}) || !strings.Contains(err.Error(), "unconfirmed effects may remain") {
		t.Fatal("recovery or uncertainty lost", result, err)
	}
}
