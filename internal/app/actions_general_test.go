package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsGeneralIndependentPlanAndMatchingApply(t *testing.T) {
	days, gb, access := 7, 10, "none"
	desiredDays, desiredGB, desiredAccess := 30, 20, "user"
	d := settings.Declaration{Actions: settings.ActionsPolicyDeclaration{CacheRetentionDays: &desiredDays, CacheSizeGB: &desiredGB, AccessLevel: &desiredAccess}}
	read := Reader{
		CacheRetention: func(repository.Repository) (settings.CacheRetentionObservation, error) {
			return settings.NewCacheRetentionObservation(days, "private")
		},
		CacheStorage: func(repository.Repository) (settings.CacheStorageObservation, error) {
			return settings.NewCacheStorageObservation(gb)
		},
		ActionsAccess: func(repository.Repository) (settings.ActionsAccessObservation, error) {
			return settings.NewActionsAccessObservation(access, "private", "User")
		},
	}
	load := func() (settings.Declaration, error) { return d, nil }
	plan, err := Plan(load, checkoutRepository, read)
	if err != nil || plan.Comparison.Managed != 3 || len(plan.Comparison.Actions) != 3 {
		t.Fatal(plan, err)
	}
	want := []string{"actions.cache_retention_days", "actions.cache_size_gb", "actions.access_level"}
	for i, c := range plan.Comparison.Actions {
		if c.Field() != want[i] {
			t.Fatal(c.Field(), want)
		}
	}
	var writes []string
	writer := Writer{ReadCacheRetention: read.CacheRetention, ReadCacheStorage: read.CacheStorage, ReadActionsAccess: read.ActionsAccess,
		CacheRetention: func(_ repository.Repository, v int) (bool, error) {
			writes = append(writes, "retention")
			days = v
			return true, nil
		},
		CacheStorage: func(_ repository.Repository, v int) (bool, error) {
			writes = append(writes, "storage")
			gb = v
			return true, nil
		},
		ActionsAccess: func(_ repository.Repository, v string) (bool, error) {
			writes = append(writes, "access")
			access = v
			return true, nil
		},
	}
	for i := 0; i < 2; i++ {
		result, err := Apply(load, checkoutRepository, writer)
		if err != nil || !result.Completed() {
			t.Fatal(result, err)
		}
	}
	if !reflect.DeepEqual(writes, []string{"retention", "storage", "access"}) {
		t.Fatal(writes)
	}
	d.Actions = settings.ActionsPolicyDeclaration{CacheSizeGB: &desiredGB}
	plan, err = Plan(load, checkoutRepository, Reader{CacheStorage: read.CacheStorage})
	if err != nil || plan.Comparison.Managed != 1 || len(plan.Comparison.Actions) != 0 {
		t.Fatal(plan, err)
	}
}

func TestActionsGeneralFailureClassificationAndStopsLaterWrites(t *testing.T) {
	for _, resource := range []string{"retention", "storage", "access"} {
		for _, earlier := range []bool{false, true} {
			for _, stage := range []string{"read", "preflight", "fresh-read", "write", "race-noop"} {
				t.Run(resource+"/"+stage+"/"+map[bool]string{true: "earlier", false: "first"}[earlier], func(t *testing.T) {
					days, gb, level := 30, 20, "user"
					d := settings.Declaration{}
					switch resource {
					case "retention":
						d.Actions.CacheRetentionDays = &days
					case "storage":
						d.Actions.CacheSizeGB = &gb
					case "access":
						d.Actions.AccessLevel = &level
					}
					earlierValue := "read"
					if earlier {
						d.Actions.DefaultWorkflowPermissions = &earlierValue
					}
					cause := errors.New("unavailable")
					written := 0
					writeResult := func() (bool, error) {
						written++
						switch stage {
						case "fresh-read":
							return false, &workflowReadFailure{cause}
						case "race-noop":
							return false, nil
						default:
							return false, cause
						}
					}
					writer := Writer{WorkflowPermissions: func(repository.Repository, settings.ActionsPolicyDeclaration) (bool, error) { return true, nil },
						ReadCacheRetention: func(repository.Repository) (settings.CacheRetentionObservation, error) {
							if stage == "read" {
								return settings.CacheRetentionObservation{}, cause
							}
							if stage == "preflight" {
								return settings.CacheRetentionObservation{}, nil
							}
							return settings.NewCacheRetentionObservation(7, "public")
						},
						ReadCacheStorage: func(repository.Repository) (settings.CacheStorageObservation, error) {
							if stage == "read" {
								return settings.CacheStorageObservation{}, cause
							}
							if stage == "preflight" {
								return settings.CacheStorageObservation{}, nil
							}
							return settings.NewCacheStorageObservation(10)
						},
						ReadActionsAccess: func(repository.Repository) (settings.ActionsAccessObservation, error) {
							if stage == "read" {
								return settings.ActionsAccessObservation{}, cause
							}
							if stage == "preflight" {
								return settings.ActionsAccessObservation{}, nil
							}
							return settings.NewActionsAccessObservation("none", "private", "User")
						},
						CacheRetention: func(repository.Repository, int) (bool, error) { return writeResult() }, CacheStorage: func(repository.Repository, int) (bool, error) { return writeResult() }, ActionsAccess: func(repository.Repository, string) (bool, error) { return writeResult() },
					}
					result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, writer)
					if stage == "race-noop" {
						if err != nil || result.Operations[len(result.Operations)-1].Status != OperationUnchanged {
							t.Fatal(result, err)
						}
						return
					}
					want := ApplyFailed
					if earlier {
						want = ApplyPartialFailure
					}
					status := OperationFailedRead
					if stage == "write" {
						status = OperationFailedUnknown
					}
					if err == nil || result.Outcome != want || result.Operations[len(result.Operations)-1].Status != status {
						t.Fatal(result, err)
					}
					if (stage == "read" || stage == "preflight") && written != 0 {
						t.Fatal("write after failed prerequisite")
					}
				})
			}
		}
	}
}
