package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestValidateLoadsDeclaration(t *testing.T) {
	empty, disabled := "", false
	for _, want := range []settings.Declaration{
		{},
		{Repository: settings.RepositoryDeclaration{Description: &empty, HasIssues: &disabled}},
	} {
		calls := 0
		got, err := Validate(func() (settings.Declaration, error) {
			calls++
			return want, nil
		})
		if err != nil || !reflect.DeepEqual(got, want) || calls != 1 {
			t.Fatalf("declaration=%+v error=%v loads=%d, want %+v and one load", got, err, calls, want)
		}
	}
}

func TestValidateLoadFailure(t *testing.T) {
	failure := errors.New("validate .github/settings.yml: declaration rejected")
	calls := 0
	got, err := Validate(func() (settings.Declaration, error) {
		calls++
		return settings.Declaration{}, failure
	})
	if err != failure || !reflect.DeepEqual(got, settings.Declaration{}) || calls != 1 {
		t.Fatalf("declaration=%+v error=%v loads=%d", got, err, calls)
	}
}
