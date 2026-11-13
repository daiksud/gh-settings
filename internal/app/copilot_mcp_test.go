package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestCopilotMCPPlan(t *testing.T) {
	for _, tc := range []struct {
		name            string
		before, desired bool
		mode            string
		changes         int
	}{
		{"disable", true, false, "", 1}, {"enable", false, true, "", 1}, {"matching false", false, false, "", 0}, {"denied", false, true, "denied", 0}, {"unread", false, true, "unread", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			result, err := Plan(func() (settings.Declaration, error) {
				return settings.Declaration{CopilotMCP: &tc.desired}, nil
			}, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Reader{CopilotMCP: func(repo repository.Repository) (settings.CopilotMCPObservation, error) {
				reads++
				if repo.Owner != "owner" || repo.Name != "current" {
					t.Fatal("wrong target")
				}
				if tc.mode == "denied" {
					return settings.CopilotMCPObservation{}, errors.New("denied")
				}
				if tc.mode == "unread" {
					return settings.CopilotMCPObservation{}, nil
				}
				return settings.NewCopilotMCPObservation(tc.before), nil
			}})
			if (err != nil) != (tc.mode != "") || reads != 1 || len(result.Comparison.CopilotMCP) != tc.changes {
				t.Fatal(result, err, reads)
			}
			if tc.mode != "" {
				if result.Comparison.Managed != 0 || result.Comparison.HasChanges() {
					t.Fatal("partial plan exposed")
				}
				return
			}
			if result.Comparison.Managed != 1 || result.Comparison.HasChanges() != (tc.changes > 0) {
				t.Fatal(result)
			}
			if tc.changes > 0 {
				before, after := result.Comparison.CopilotMCP[0].Values()
				if before != tc.before || after != tc.desired {
					t.Fatal("wrong boolean comparison")
				}
			}
		})
	}
	_, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{}, nil }, func() (repository.Repository, error) {
		t.Fatal("resolved omitted setting")
		return repository.Repository{}, nil
	}, Reader{})
	if err != nil {
		t.Fatal(err)
	}
}

type mcpReadFailure struct{ error }

func (*mcpReadFailure) ReadFailedBeforeWrite() {}
func TestCopilotMCPApply(t *testing.T) {
	target := func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}
	for _, tc := range []struct {
		name           string
		desired, wrote bool
		err            error
		outcome        ApplyOutcome
		status         ApplyOperationStatus
	}{
		{"changed false", false, true, nil, ApplyCompleted, OperationCompleted},
		{"changed true", true, true, nil, ApplyCompleted, OperationCompleted},
		{"matching false", false, false, nil, ApplyMatchingLabels, OperationUnchanged},
		{"preread failure", false, false, &mcpReadFailure{errors.New("denied")}, ApplyFailed, OperationFailedRead},
		{"unknown write", true, false, errors.New("write may have applied"), ApplyFailed, OperationFailedUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			result, err := Apply(func() (settings.Declaration, error) {
				return settings.Declaration{CopilotMCP: &tc.desired}, nil
			}, target, Writer{CopilotMCP: func(repo repository.Repository, desired bool) (bool, error) {
				calls++
				if desired != tc.desired || repo.Name != "current" {
					t.Fatal("wrong intent/target")
				}
				return tc.wrote, tc.err
			}})
			if (err != nil) != (tc.err != nil) || calls != 1 || result.Outcome != tc.outcome || len(result.Operations) != 1 || result.Operations[0].Status != tc.status || result.Operations[0].Kind != CopilotMCPUpdate {
				t.Fatal(result, err, calls)
			}
		})
	}
	enabled := true
	description := "changed"
	var calls []string
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}, CopilotMCP: &enabled, Security: settings.SecurityDeclaration{PrivateVulnerabilityReporting: &enabled}}, nil
	}, target, Writer{
		Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
			calls = append(calls, "repository")
			return nil
		},
		CopilotMCP: func(repository.Repository, bool) (bool, error) {
			calls = append(calls, "copilot")
			return false, errors.New("uncertain update")
		},
		ReadPrivateReporting: func(repository.Repository) (settings.PrivateReportingObservation, error) {
			t.Fatal("later resource read after failure")
			return settings.PrivateReportingObservation{}, nil
		},
	})
	if err == nil || result.Outcome != ApplyPartialFailure || len(calls) != 2 || calls[0] != "repository" || calls[1] != "copilot" || result.Operations[0].Status != OperationCompleted {
		t.Fatal(result, err, calls)
	}
	result, err = Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}}, nil
	}, target, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) error { return nil }, CopilotMCP: func(repository.Repository, bool) (bool, error) { t.Fatal("called omitted setting"); return false, nil }})
	if err != nil || result.Outcome != ApplyCompleted || len(result.Operations) != 1 {
		t.Fatal(result, err)
	}
	archived := true
	_, err = Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, CopilotMCP: &enabled}, nil
	}, target, Writer{ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
		return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
	}, CopilotMCP: func(repository.Repository, bool) (bool, error) { t.Fatal("updated archived target"); return false, nil }})
	if err == nil {
		t.Fatal("allowed incompatible archived-target update")
	}
}

func TestCopilotMCPExport(t *testing.T) {
	for _, tc := range []struct {
		name             string
		include, enabled bool
		mode             string
	}{
		{"enabled", true, true, ""}, {"disabled", true, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := exportFixtureReader(t, settings.RepositoryState{}, func(string, repository.Repository) error { return nil })
			reads, saves := 0, 0
			reader.CopilotMCP = func(repo repository.Repository) (settings.CopilotMCPObservation, error) {
				reads++
				if repo.Name != "current" {
					t.Fatal("wrong target")
				}
				if tc.mode == "denied" {
					return settings.CopilotMCPObservation{}, errors.New("denied")
				}
				if tc.mode == "unread" {
					return settings.CopilotMCPObservation{}, nil
				}
				return settings.NewCopilotMCPObservation(tc.enabled), nil
			}
			err := Export(ExportCheckout{Repository: func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saves++
					if (d.CopilotMCP != nil) != tc.include || tc.include && *d.CopilotMCP != tc.enabled {
						t.Fatal(d)
					}
					return nil
				}, nil
			}}, reader)
			if (err != nil) != (tc.mode != "") || reads != map[bool]int{false: 0, true: 1}[tc.include] || (saves == 1) != (tc.mode == "") {
				t.Fatal(err, reads, saves)
			}
		})
	}
}
