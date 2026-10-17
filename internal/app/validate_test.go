package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func TestValidateReadsOnlySettingsFile(t *testing.T) {
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
			path := filepath.Join(root, workspace.SettingsPath)
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
			t.Chdir(work) // No origin is configured; validation stays local.
			empty, disabled := "", false
			want := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, HasIssues: &disabled}}
			got, err := Validate()
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

func TestValidateErrors(t *testing.T) {
	for _, test := range []struct {
		name, input, want string
		outsideGit        bool
	}{
		{name: "missing file", want: "read .github/settings.yml"},
		{name: "invalid document", input: "settings: {}\n", want: "validate .github/settings.yml: missing schemaVersion"},
		{name: "outside Git", outsideGit: true, want: "locate .github/settings.yml: current directory is not in a Git repository"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if !test.outsideGit {
				if output, err := exec.Command("git", "init", "--quiet", root).CombinedOutput(); err != nil {
					t.Fatalf("git init: %v: %s", err, output)
				}
			}
			if test.input != "" {
				if err := os.Mkdir(filepath.Join(root, ".github"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, workspace.SettingsPath), []byte(test.input), 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Chdir(root)
			got, err := Validate()
			if err == nil || !strings.Contains(err.Error(), test.want) || !reflect.DeepEqual(got, settings.Declaration{Repository: settings.RepositoryDeclaration{}}) {
				t.Fatalf("declaration=%+v error=%v, want zero declaration and %q", got, err, test.want)
			}
		})
	}
}
