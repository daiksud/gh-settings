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
