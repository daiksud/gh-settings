package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func writebackFixture(t *testing.T, input string) (string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, SettingsPath)
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	return root, path
}

func TestCreatedIDWritebackRetainsExistingValuesCommentsAndEarlierIDs(t *testing.T) {
	input := "# consumer configuration\nschemaVersion: 1\nsettings:\n  repository: {description: keep}\n  labels:\n    - name: old\n      id: 4\n      color: abcdef\n    - name: new # keep this comment\n      color: '123456'\n    - {name: later, color: fedcba}\n"
	root, path := writebackFixture(t, input)
	record := createdIDRecorder(root, []byte(input))
	if err := record("labels", "new", 17); err != nil {
		t.Fatal(err)
	}
	if err := record("labels", "later", 18); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range []string{"# consumer configuration", "# keep this comment"} {
		if !strings.Contains(string(data), comment) {
			t.Fatalf("lost comment: %s", data)
		}
	}
	var document struct {
		Settings struct {
			Repository map[string]string
			Labels     []struct {
				Name  string
				ID    int64
				Color string
			}
		}
	}
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document.Settings.Repository["description"] != "keep" {
		t.Fatal(document)
	}
	for i, want := range []int64{4, 17, 18} {
		if document.Settings.Labels[i].ID != want {
			t.Fatal(document.Settings.Labels)
		}
	}
	if document.Settings.Labels[0].Color != "abcdef" || document.Settings.Labels[1].Color != "123456" || document.Settings.Labels[2].Color != "fedcba" {
		t.Fatal(document.Settings.Labels)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("file mode changed: %v %v", info, err)
	}
}

func TestCreatedIDWritebackFailurePreservesSettings(t *testing.T) {
	for _, tc := range []struct {
		name, input, collection, object string
		id                              int64
		edit                            bool
	}{
		{"existing ID", "settings: {labels: [{name: x, id: 4}]}\n", "labels", "x", 7, false},
		{"missing object", "settings: {labels: [{name: x}]}\n", "labels", "absent", 7, false},
		{"ambiguous object", "settings: {labels: [{name: x}, {name: x}]}\n", "labels", "x", 7, false},
		{"invalid ID", "settings: {labels: [{name: x}]}\n", "labels", "x", 0, false},
		{"wrong collection", "settings: {labels: [{name: x}]}\n", "repository", "x", 7, false},
		{"concurrent edit", "settings: {labels: [{name: x}]}\n", "labels", "x", 7, true},
		{"aliased collection", "value: &items [{name: x}]\nsettings: {labels: *items}\n", "labels", "x", 7, false},
		{"aliased entry", "value: &item {name: x}\nsettings: {labels: [*item]}\n", "labels", "x", 7, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, path := writebackFixture(t, tc.input)
			record := createdIDRecorder(root, []byte(tc.input))
			want := tc.input
			if tc.edit {
				want += "# concurrent user edit\n"
				if err := os.WriteFile(path, []byte(want), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := record(tc.collection, tc.object, tc.id); err == nil {
				t.Fatal("invalid writeback succeeded")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != want {
				t.Fatalf("settings changed: %s %v", data, err)
			}
		})
	}
}

func TestCreatedIDWritebackRejectsSymlinks(t *testing.T) {
	input := "settings: {labels: [{name: x}]}\n"
	for _, directory := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "directory"}[directory], func(t *testing.T) {
			root, path := writebackFixture(t, input)
			outside := t.TempDir()
			if directory {
				if err := os.Rename(filepath.Join(root, ".github"), filepath.Join(outside, "real")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(outside, "real"), filepath.Join(root, ".github")); err != nil {
					t.Fatal(err)
				}
			} else {
				target := filepath.Join(outside, "settings.yml")
				if err := os.Rename(path, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			}
			if err := createdIDRecorder(root, []byte(input))("labels", "x", 7); err == nil {
				t.Fatal("symlink writeback succeeded")
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != input {
				t.Fatalf("linked target changed: %s %v", data, err)
			}
		})
	}
}
