package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestReadConfigAtRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".github", "settings.yml")
	input := []byte("version: daiksud/gh-settings@v0\nrepository: {about: {description: ''}, settings: {general: {features: {issues: false}}}}\n")
	if err := os.WriteFile(path, input, 0600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	empty, disabled := "", false
	want := uimodel.Repository{About: &uimodel.About{Description: &empty}, Settings: &uimodel.RepositorySettings{General: &uimodel.General{Features: uimodel.Features{Issues: &disabled}}}}
	got, err := ReadConfig(root)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("declaration=%+v error=%v, want %+v", got, err, want)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, input) {
		t.Fatalf("source changed: %q error=%v", after, err)
	}
}

func TestReadConfigErrors(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"missing file", "", "read .github/settings.yml"},
		{"invalid YAML", "repository: {}\n", "validate .github/settings.yml: missing version"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if test.input != "" {
				if err := os.Mkdir(filepath.Join(root, ".github"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, ".github", "settings.yml"), []byte(test.input), 0600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := ReadConfig(root)
			if err == nil || !strings.Contains(err.Error(), test.want) || !reflect.DeepEqual(got, uimodel.Repository{}) {
				t.Fatalf("declaration=%+v error=%v, want zero declaration and %q", got, err, test.want)
			}
		})
	}
}

func TestWriteConfigAtRoot(t *testing.T) {
	tests := []struct {
		name    string
		present bool
	}{{name: "create"}, {name: "replace", present: true}}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, ".github", "settings.yml")
			if test.present {
				if err := os.Mkdir(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("existing configuration\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			nested := filepath.Join(root, "nested")
			if err := os.Mkdir(nested, 0700); err != nil {
				t.Fatal(err)
			}
			t.Chdir(nested)
			input := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {issues: false}}}}\n")
			if err := WriteConfig(root, input); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, input) {
				t.Fatalf("saved=%q error=%v, want %q", got, err, input)
			}
			if _, err := os.Lstat(filepath.Join(nested, ".github")); !os.IsNotExist(err) {
				t.Fatalf("subdirectory destination exists: %v", err)
			}
		})
	}
}

func TestWriteConfigRejectsUnsafeDestination(t *testing.T) {
	tests := []struct{ name, want string }{
		{"parent symlink", ".github must be a directory"},
		{"file symlink", ".github/settings.yml must be a regular file"},
		{"directory", ".github/settings.yml must be a regular file"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			sentinel := filepath.Join(outside, "settings.yml")
			if err := os.WriteFile(sentinel, []byte("external configuration\n"), 0600); err != nil {
				t.Fatal(err)
			}
			github := filepath.Join(root, ".github")
			path := filepath.Join(github, "settings.yml")
			observed := path
			switch test.name {
			case "parent symlink":
				if err := os.Symlink(outside, github); err != nil {
					t.Fatal(err)
				}
				observed = github
			case "file symlink":
				if err := os.Mkdir(github, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(sentinel, path); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.Lstat(observed)
			if err != nil {
				t.Fatal(err)
			}
			disabled := false
			for _, save := range []func() error{
				func() error {
					return WriteConfig(root, []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {general: {features: {issues: false}}}}\n"))
				},
				func() error {
					return saveDeclaration(root, uimodel.Repository{Settings: &uimodel.RepositorySettings{General: &uimodel.General{Features: uimodel.Features{Issues: &disabled}}}})
				},
			} {
				err = save()
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("error=%v, want %q", err, test.want)
				}
			}
			after, err := os.Lstat(observed)
			if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
				t.Fatalf("destination changed: before=%v after=%v error=%v", before, after, err)
			}
			data, err := os.ReadFile(sentinel)
			if err != nil || string(data) != "external configuration\n" {
				t.Fatalf("external file changed: %q error=%v", data, err)
			}
		})
	}
}
