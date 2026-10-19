package app

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestApplyDeclaredSettings(t *testing.T) {
	input := "schemaVersion: 1\nsettings:\n  repository:\n    description: ''\n    has_issues: false\n"
	path := appCheckout(t, input, true)
	t.Setenv("GH_REPO", "other/repository")
	empty, disabled := "", false
	want := settings.RepositoryDeclaration{Description: &empty, HasIssues: &disabled}
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	calls := 0
	got, err := Apply(checkoutDeclaration, Writer{Repository: func(repo repository.Repository, desired settings.RepositoryDeclaration) error {
		calls++
		if repo != target || !reflect.DeepEqual(desired, want) {
			t.Fatalf("target=%+v declaration=%+v, want %+v %+v", repo, desired, target, want)
		}
		return nil
	}})
	if err != nil || got != (ApplyResult{Applied: true}) || calls != 1 {
		t.Fatalf("result=%+v error=%v writes=%d, want applied and one write", got, err, calls)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, []byte(input)) {
		t.Fatalf("settings changed: %q error=%v", after, err)
	}
}

func TestApplyNoManagedSettings(t *testing.T) {
	for _, test := range []struct{ name, input string }{
		{"version only", "schemaVersion: 1\n"},
		{"empty settings", "schemaVersion: 1\nsettings: {}\n"},
		{"empty repository", "schemaVersion: 1\nsettings:\n  repository: {}\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			appCheckout(t, test.input, false) // No origin: a no-op must finish before target resolution.
			calls := 0
			got, err := Apply(checkoutDeclaration, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
				calls++
				return nil
			}})
			if err != nil || got != (ApplyResult{}) || calls != 0 {
				t.Fatalf("result=%+v error=%v writes=%d, want local no-op", got, err, calls)
			}
		})
	}
}

func TestApplyValidatesBeforeWrite(t *testing.T) {
	loadError := errors.New("validate .github/settings.yml: declaration rejected")
	loads := 0
	got, err := Apply(func() (settings.Declaration, error) {
		loads++
		disabled := false
		return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}}, loadError
	}, Writer{})
	if err != loadError || got != (ApplyResult{}) || loads != 1 {
		t.Fatalf("result=%+v error=%v loads=%d, want original validation failure before target/write", got, err, loads)
	}
}

func TestApplyTargetFailureBeforeWrite(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n", false)
	calls := 0
	got, err := Apply(checkoutDeclaration, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
		calls++
		return nil
	}})
	if err == nil || !strings.Contains(err.Error(), "a GitHub origin remote is required") || got != (ApplyResult{}) || calls != 0 {
		t.Fatalf("result=%+v error=%v writes=%d, want target failure before write", got, err, calls)
	}
}

func TestApplyWriteFailure(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n", true)
	writeError := errors.New("write unavailable")
	calls := 0
	got, err := Apply(checkoutDeclaration, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
		calls++
		return writeError
	}})
	if !errors.Is(err, writeError) || got != (ApplyResult{}) || calls != 1 {
		t.Fatalf("result=%+v error=%v writes=%d, want write error without applied status", got, err, calls)
	}
}
