package workspace

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestCheckoutDeclaration(t *testing.T) {
	for _, test := range []struct {
		name, rootName string
		nested         bool
	}{
		{"nested checkout", "repo", true},
		{"trailing-space root", "repo ", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), test.rootName)
			if err := os.MkdirAll(filepath.Join(root, ".github"), 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, SettingsPath)
			input := []byte("schemaVersion: 1\nsettings:\n  repository:\n    description: ''\n    has_issues: false\n")
			if err := os.WriteFile(path, input, 0600); err != nil {
				t.Fatal(err)
			}
			if output, err := exec.Command("git", "init", "--quiet", root).CombinedOutput(); err != nil {
				t.Fatalf("git init: %v: %s", err, output)
			}
			work := root
			if test.nested {
				work = filepath.Join(root, "nested")
				if err := os.Mkdir(work, 0700); err != nil {
					t.Fatal(err)
				}
			}
			t.Chdir(t.TempDir()) // The explicit checkout, not process cwd, owns the declaration.
			empty, disabled := "", false
			want := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, HasIssues: &disabled}}
			got, err := NewCheckout(work).Declaration()
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("declaration=%+v error=%v, want %+v", got, err, want)
			}
			unchanged, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(unchanged, input) {
				t.Fatalf("settings changed: %q, error=%v", unchanged, err)
			}
		})
	}
}

func TestCheckoutDeclarationOutsideGit(t *testing.T) {
	got, err := NewCheckout(t.TempDir()).Declaration()
	if err == nil || !strings.Contains(err.Error(), "locate .github/settings.yml: current directory is not in a Git repository") || !reflect.DeepEqual(got, settings.Declaration{}) {
		t.Fatalf("declaration=%+v error=%v", got, err)
	}
}

func TestCheckoutRepositoryUsesItsDirectory(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "0")
	var checkouts []Checkout
	for _, name := range []string{"first", "second"} {
		root := filepath.Join(t.TempDir(), name+" ")
		nested := filepath.Join(root, "nested")
		if err := os.MkdirAll(nested, 0700); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{
			{"init", "--quiet", root},
			{"-C", root, "remote", "add", "origin", "https://github.com/owner/" + name + ".git"},
		} {
			if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
				t.Fatalf("git: %v: %s", err, output)
			}
		}
		checkouts = append(checkouts, NewCheckout(nested))
	}
	t.Chdir(checkouts[0].directory)
	for _, index := range []int{1, 0, 1} {
		got, err := checkouts[index].Repository()
		want := []string{"first", "second"}[index]
		if err != nil || got.Host != "github.com" || got.Owner != "owner" || got.Name != want {
			t.Fatalf("checkout %d selected %+v, error=%v, want owner/%s", index, got, err, want)
		}
	}
	// An outside directory must not borrow the process cwd's valid checkout.
	if got, err := NewCheckout(t.TempDir()).Repository(); err == nil || !strings.Contains(err.Error(), "a GitHub origin remote is required") || got.Host != "" {
		t.Fatalf("outside checkout selected %+v, error=%v", got, err)
	}
}

func TestCheckoutPrepareSaveBindsResolvedRoot(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "init", "--quiet", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	github := filepath.Join(root, ".github")
	if err := os.Mkdir(github, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(github, "settings.yml")
	before := "existing configuration\n"
	if err := os.WriteFile(path, []byte(before), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	save, err := NewCheckout(nested).PrepareSave()
	if err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != before {
		t.Fatalf("destination changed during resolution: %q %v", data, err)
	}
	other := t.TempDir()
	t.Chdir(other)
	disabled := false
	if err := save(settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}}); err != nil {
		t.Fatal(err)
	}
	want := "schemaVersion: 1\nsettings:\n    repository:\n        has_issues: false\n"
	if data, err := os.ReadFile(path); err != nil || string(data) != want {
		t.Fatalf("saved=%q error=%v, want %q", data, err, want)
	}
	if _, err := os.Lstat(filepath.Join(other, ".github")); !os.IsNotExist(err) {
		t.Fatalf("unrelated destination exists: %v", err)
	}
}

func TestCheckoutPrepareSaveOutsideGit(t *testing.T) {
	save, err := NewCheckout(t.TempDir()).PrepareSave()
	if save != nil || err == nil || !strings.Contains(err.Error(), "locate .github/settings.yml:") {
		t.Fatalf("save present=%v error=%v", save != nil, err)
	}
}
