package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestPullRequestLimitsPlanAndApply(t *testing.T) {
	enabled := false
	maximum := 2
	desired := settings.PullRequestLimitsDeclaration{Enabled: &enabled, Maximum: &maximum}
	load := func() (settings.Declaration, error) { return settings.Declaration{PullRequestLimits: desired}, nil }
	target := func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}
	result, err := Plan(load, target, Reader{PullRequestLimits: func(repo repository.Repository) (settings.PullRequestLimitsObservation, error) {
		return settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: true, Maximum: 1, IncludeDrafts: true})
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
			if d.Enabled == nil || *d.Enabled || d.IncludeDrafts != nil || *d.Maximum != 2 {
				t.Fatal(d)
			}
			return tc.wrote, tc.err
		}})
		if (err != nil) != (tc.err != nil) || len(r.Operations) != 1 || r.Operations[0].Kind != PullRequestLimitsUpdate || r.Operations[0].Status != tc.status {
			t.Fatal(r, err)
		}
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
