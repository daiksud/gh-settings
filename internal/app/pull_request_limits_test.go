package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPullRequestLimitsPlanAndApply(t *testing.T) {
	enabled := true
	maximum := 2
	desired := settings.PullRequestLimitsDeclaration{Enabled: &enabled, Maximum: &maximum}
	load := func() (settings.Declaration, error) { return settings.Declaration{PullRequestLimits: desired}, nil }
	target := func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}
	result, err := Plan(load, target, Reader{PullRequestLimits: func(repo repository.Repository) (settings.PullRequestLimitsObservation, error) {
		return settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: false, Maximum: 1, IncludeDrafts: true})
	}})
	if err != nil || result.Comparison.Managed != 2 || len(result.Comparison.PullRequestLimits) != 2 {
		t.Fatal(result, err)
	}
	for _, tc := range []struct {
		err    error
		wrote  bool
		status ApplyOperationStatus
	}{{nil, true, OperationCompleted}, {nil, false, OperationUnchanged}, {errors.New("uncertain"), false, OperationFailedUnknown}, {&mcpReadFailure{errors.New("denied")}, false, OperationFailedRead}} {
		r, err := Apply(load, target, Writer{PullRequestLimits: func(repo repository.Repository, d settings.PullRequestLimitsDeclaration) (bool, error) {
			if d.Enabled == nil || !*d.Enabled || d.IncludeDrafts != nil || d.Maximum == nil || *d.Maximum != 2 {
				t.Fatal(d)
			}
			return tc.wrote, tc.err
		}})
		if (err != nil) != (tc.err != nil) || len(r.Operations) != 1 || r.Operations[0].Kind != PullRequestLimitsUpdate || r.Operations[0].Status != tc.status {
			t.Fatal(r, err)
		}
	}
}

func TestExportPullRequestLimitControlsFollowObservedOrRetainedSwitch(t *testing.T) {
	yes, no, maximum := true, false, 7
	active := settings.PullRequestLimitsDeclaration{Enabled: &yes, Maximum: &maximum, IncludeDrafts: &no}
	disabled := settings.PullRequestLimitsDeclaration{Enabled: &no}
	unspecified := settings.PullRequestLimitsDeclaration{}

	for _, tc := range []struct {
		name        string
		prior, want settings.PullRequestLimitsDeclaration
		observed    *bool
		listReads   int
	}{
		{"disabled since prior export", active, disabled, &no, 0},
		{"enabled since prior export", disabled, active, &yes, 1},
		{"unreadable with prior disabled", disabled, disabled, nil, 0},
		{"unreadable with prior enabled", active, active, nil, 1},
		{"unreadable on first export", unspecified, unspecified, nil, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior := settings.Declaration{PullRequestLimits: tc.prior}
			if tc.prior.Enabled != nil && *tc.prior.Enabled {
				list, _ := settings.NewPullRequestExemptionsDeclaration([]string{"previous-user"})
				prior.PullRequestExemptions = &list
			}
			listReads := 0
			reader := Reader{
				PullRequestLimits: func(repository.Repository) (settings.PullRequestLimitsObservation, error) {
					if tc.observed == nil {
						return settings.PullRequestLimitsObservation{}, errors.New("unavailable")
					}
					return settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: *tc.observed, Maximum: maximum})
				},
				PullRequestExemptions: func(repository.Repository) (settings.PullRequestExemptionsObservation, error) {
					listReads++
					return settings.NewPullRequestExemptionsObservation(nil)
				},
			}
			var saved settings.Declaration
			err := Export(ExportCheckout{Existing: prior, Repository: checkoutRepository, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error { saved = d; return nil }, nil
			}}, reader)
			if err != nil || !reflect.DeepEqual(saved.PullRequestLimits, tc.want) || listReads != tc.listReads {
				t.Fatalf("limits=%+v, exemption reads=%d, err=%v", saved.PullRequestLimits, listReads, err)
			}
			list := saved.PullRequestExemptions
			if (list != nil) != (tc.listReads == 1) || list != nil && len(list.Logins()) != 0 {
				t.Fatalf("unexpected exported exemption list: %v", list)
			}
		})
	}
}

func TestPullRequestLimitsPartialAndArchive(t *testing.T) {
	enabled, archived := true, true
	description := "updated"
	target := func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}, PullRequestLimits: settings.PullRequestLimitsDeclaration{Enabled: &enabled}}, nil
	}, target, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { return true, nil }, PullRequestLimits: func(repository.Repository, settings.PullRequestLimitsDeclaration) (bool, error) {
		return false, errors.New("uncertain")
	}})
	if err == nil || result.Outcome != ApplyPartialFailure || len(result.Operations) != 2 || result.Operations[0].Status != OperationCompleted || result.Operations[1].Status != OperationFailedUnknown {
		t.Fatal(result, err)
	}
	_, err = Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, PullRequestLimits: settings.PullRequestLimitsDeclaration{Enabled: &enabled}}, nil
	}, target, Writer{ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
		return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
	}, PullRequestLimits: func(repository.Repository, settings.PullRequestLimitsDeclaration) (bool, error) {
		t.Fatal("archive conflict wrote cap")
		return false, nil
	}})
	if err == nil {
		t.Fatal("archive conflict accepted")
	}
}
