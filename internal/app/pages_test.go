package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"testing"
)

func TestPagesPlanOmissionAndPartialFailures(t *testing.T) {
	domain := "example.com"
	o, _ := settings.NewPagesObservation(settings.PagesState{BuildType: "workflow", CNAME: &domain, Public: true})
	d := settings.Declaration{Pages: settings.PagesDeclaration{CNAME: &settings.PagesDomain{}}}
	load := func() (settings.Declaration, error) { return d, nil }
	p, err := Plan(load, checkoutRepository, Reader{Pages: func(repository.Repository) (settings.PagesObservation, error) { return o, nil }})
	if err != nil || p.Comparison.Managed != 1 || len(p.Comparison.Pages) != 1 {
		t.Fatal(p, err)
	}
	for _, earlier := range []bool{false, true} {
		for _, failure := range []string{"read", "prewrite", "write"} {
			value := "changed"
			d.Repository.Description = nil
			if earlier {
				d.Repository.Description = &value
			}
			r, err := Apply(load, checkoutRepository, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) error { return nil }, ReadPages: func(repository.Repository) (settings.PagesObservation, error) {
				if failure == "read" {
					return settings.PagesObservation{}, errors.New("unavailable")
				}
				return o, nil
			}, Pages: func(repository.Repository, settings.PagesDeclaration) (bool, error) {
				if failure == "prewrite" {
					return false, &workflowReadFailure{errors.New("fresh read")}
				}
				return false, errors.New("uncertain update")
			}})
			want := ApplyFailed
			if earlier {
				want = ApplyPartialFailure
			}
			status := OperationFailedRead
			if failure == "write" {
				status = OperationFailedUnknown
			}
			if err == nil || r.Outcome != want || r.Operations[len(r.Operations)-1].Status != status {
				t.Fatal(earlier, failure, r, err)
			}
		}
	}
	d = settings.Declaration{}
	if _, err := Plan(load, func() (repository.Repository, error) {
		t.Fatal("unmanaged resolves target")
		return repository.Repository{}, nil
	}, Reader{}); err != nil {
		t.Fatal(err)
	}
}
func TestPagesBetweenExplicitUnarchiveAndFinalArchive(t *testing.T) {
	for _, archived := range []bool{false, true} {
		d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, Pages: settings.PagesDeclaration{CNAME: &settings.PagesDomain{}}}
		domain := "example.com"
		o, _ := settings.NewPagesObservation(settings.PagesState{BuildType: "workflow", CNAME: &domain, Public: true})
		events := []string{}
		state := settings.ArchiveState{ID: 123, FullName: "owner/current", Archived: !archived}
		r, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) {
			return repository.Repository{Owner: "owner", Name: "current"}, nil
		}, Writer{ReadArchive: func(repository.Repository) (settings.ArchiveState, error) { return state, nil }, Archive: func(_ repository.Repository, _ settings.ArchiveState, v bool) error {
			if v {
				events = append(events, "archive")
			} else {
				events = append(events, "unarchive")
			}
			state.Archived = v
			return nil
		}, ReadPages: func(repository.Repository) (settings.PagesObservation, error) { return o, nil }, Pages: func(repository.Repository, settings.PagesDeclaration) (bool, error) {
			events = append(events, "pages")
			return true, nil
		}})
		want := []string{"pages", "archive"}
		if !archived {
			want = []string{"unarchive", "pages"}
		}
		if err != nil || !r.Completed() || !reflect.DeepEqual(events, want) {
			t.Fatal(events, r, err)
		}
	}
}
