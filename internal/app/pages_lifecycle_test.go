package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type confirmedPagesCreation struct{ error }

func (*confirmedPagesCreation) PagesCreationCompleted() {}
func (e *confirmedPagesCreation) Unwrap() error         { return e.error }

func TestPagesConfirmedCreationSurvivesOptionalFailure(t *testing.T) {
	mode := "workflow"
	d := settings.Declaration{Pages: settings.PagesDeclaration{BuildType: &mode, CNAME: &settings.PagesDomain{}}}
	for _, prewrite := range []bool{false, true} {
		var cause error = errors.New("optional write uncertain")
		if prewrite {
			cause = &workflowReadFailure{errors.New("optional fresh read failed")}
		}
		result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
			ReadPages: func(repository.Repository) (settings.PagesObservation, error) {
				return settings.NewAbsentPagesObservation(), nil
			},
			Pages: func(repository.Repository, settings.PagesDeclaration) (bool, error) {
				return true, &confirmedPagesCreation{cause}
			},
		})
		if err == nil || result.Outcome != ApplyPartialFailure || len(result.Operations) != 3 || result.Operations[1].Kind != PagesCreate || result.Operations[1].Status != OperationCompleted {
			t.Fatal(result, err)
		}
		want := OperationFailedUnknown
		if prewrite {
			want = OperationFailedRead
		}
		if result.Operations[2].Kind != PagesUpdate || result.Operations[2].Status != want {
			t.Fatal(result)
		}
	}
}
