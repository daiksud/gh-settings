package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func checkoutDeclaration() (settings.Declaration, error) {
	ui, err := workspace.NewCheckout(".").Declaration()
	if err != nil {
		return settings.Declaration{}, err
	}
	return github.DeclarationFromUI(ui)
}

func checkoutRepository() (repository.Repository, error) {
	return workspace.NewCheckout(".").Repository()
}

func appCheckout(t *testing.T, input string, origin bool) string {
	t.Helper()
	root := t.TempDir()
	if output, err := exec.Command("git", "init", "--quiet", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if origin {
		if output, err := exec.Command("git", "-C", root, "remote", "add", "origin", "https://github.com/owner/current.git").CombinedOutput(); err != nil {
			t.Fatalf("git origin: %v: %s", err, output)
		}
	}
	if err := os.Mkdir(filepath.Join(root, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, workspace.SettingsPath)
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return path
}

func observeBaseFixture(read func(repository.Repository) (settings.RepositoryState, error)) func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
	return func(repo repository.Repository, _ []settings.FieldID) (settings.RepositoryObservation, error) {
		state, err := read(repo)
		if err != nil {
			return settings.RepositoryObservation{}, err
		}
		return settings.NewRepositoryObservation(&state, nil, nil), nil
	}
}

func observeMergeFixture(read func(repository.Repository) (settings.MergeMethodsState, error)) func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
	return func(repo repository.Repository, _ []settings.FieldID) (settings.RepositoryObservation, error) {
		state, err := read(repo)
		if err != nil {
			return settings.RepositoryObservation{}, err
		}
		return settings.NewRepositoryObservation(nil, &state, nil), nil
	}
}
