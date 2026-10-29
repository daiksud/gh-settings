package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func appLabelDeclarations(t *testing.T, values ...settings.LabelDeclaration) settings.LabelDeclarations {
	t.Helper()
	result, err := settings.NewLabelDeclarations(values)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestLabelsEmptySelectionIsLocalNoop(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {labels: []}\n", false)
	if got, err := Plan(checkoutDeclaration, func() (repository.Repository, error) {
		t.Fatal("resolved empty selection")
		return repository.Repository{}, nil
	}, Reader{}); err != nil || got.Comparison.Managed != 0 {
		t.Fatal(got, err)
	}
	if got, err := Apply(checkoutDeclaration, func() (repository.Repository, error) {
		t.Fatal("resolved empty selection")
		return repository.Repository{}, nil
	}, Writer{}); err != nil || got.Outcome != ApplyUnmanaged {
		t.Fatal(got, err)
	}
}
func TestLabelsApplyReadsFreshStateAndSkipsUnchangedWrites(t *testing.T) {
	color, id := "abcdef", int64(7)
	labels := appLabelDeclarations(t, settings.LabelDeclaration{ID: &id, Name: "bug", Color: &color})
	reads := 0
	result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Labels: &labels}, nil }, checkoutRepository, Writer{ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
		reads++
		return settings.NewLabelObservation([]settings.LabelState{{ID: 7, Name: "BUG", Color: "abcdef"}, {ID: 8, Name: "untouched", Color: "123456"}})
	}})
	if err != nil || !result.Completed() || result.Outcome != ApplyMatchingLabels || reads != 1 {
		t.Fatal(result, err, reads)
	}
}
func TestInvalidLabelsBeforeTargetAndRemoteAccess(t *testing.T) {
	for _, input := range []string{"null", "[{name: bug}]", "[{id: 0, name: bug, color: '#abcdef'}]", "[{name: bug, color: '#abcdef'}, {name: BUG, color: '#abcdef'}]", "[{name: bug, color: false}]"} {
		appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {labels: "+input+"}", false)
		if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err == nil {
			t.Fatal(input)
		}
		if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err == nil {
			t.Fatal(input)
		}
	}
}
func TestLabelsPlanFailureDoesNotExposeEarlierComparison(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository:\n  settings: {general: {features: {issues: false}}}\n  labels: [{name: new, color: '#abcdef'}]\n", true)
	for _, failureMode := range []string{"later-page", "ambiguous", "missing"} {
		result, err := Plan(checkoutDeclaration, checkoutRepository, Reader{
			ObserveRepository: observeBaseFixture(func(repository.Repository) (settings.RepositoryState, error) {
				return settings.RepositoryState{HasIssues: true}, nil
			}),
			Labels: func(repository.Repository) (settings.LabelObservation, error) {
				if failureMode == "later-page" {
					return settings.LabelObservation{}, errors.New("later page failed")
				}
				if failureMode == "missing" {
					return settings.LabelObservation{}, nil
				}
				return settings.NewLabelObservation([]settings.LabelState{{ID: 1, Name: "bug", Color: "abcdef"}, {ID: 2, Name: "BUG", Color: "abcdef"}})
			},
		})
		var failure *PlanError
		if !errors.As(err, &failure) || failure.Stage != "read-failure" || result.Comparison.Managed != 0 || len(result.Comparison.Changes) != 0 {
			t.Fatal(result, err)
		}
	}
}
func TestMissingLabelObservationPreventsExportAndApplyWrites(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {labels: [{name: new, color: '#abcdef'}]}\n", true)
	missing := func(repository.Repository) (settings.LabelObservation, error) {
		return settings.LabelObservation{}, nil
	}
	result, err := Apply(checkoutDeclaration, checkoutRepository, Writer{ReadLabels: missing})
	if err == nil || result.Completed() || !strings.Contains(err.Error(), "missing labels observation") || !reflect.DeepEqual(result.Operations, []ApplyOperation{{Kind: LabelsRead, Status: OperationFailedRead}}) {
		t.Fatal(result, err)
	}
	read := exportFixtureReader(t, settings.RepositoryState{}, func(string, repository.Repository) error { return nil })
	read.Labels = missing
	err = Export(ExportCheckout{Repository: checkoutRepository, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(settings.Declaration) error { t.Fatal("save after missing observation"); return nil }, nil
	}}, read)
	if err == nil {
		t.Fatal("export accepted missing read")
	}
}
