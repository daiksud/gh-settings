package workspace

import (
	"os"
	"strings"
	"testing"
)

func TestUICollaboratorsDeclarationRoundtrip(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    collaborators:\n      - username: person\n        id: 123\n      - username: newcomer\n"
	value, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeConfig(value)
	if err != nil || !strings.Contains(string(data), "username: person") || !strings.Contains(string(data), "id: 123") || !strings.Contains(string(data), "username: newcomer") {
		t.Fatal(string(data), err)
	}
}

func TestCollaboratorIdentityBindingFileSafety(t *testing.T) {
	for _, mode := range []string{"success", "concurrent edit", "existing id", "alias", "merged entry"} {
		t.Run(mode, func(t *testing.T) {
			input := "# keep comment\nversion: daiksud/gh-settings@v0\nrepository: {settings: {collaborators: [{username: Person}, {username: other}]}}\n"
			if mode == "existing id" {
				input = strings.Replace(input, "username: Person", "username: Person, id: 7", 1)
			}
			if mode == "alias" {
				input = "entry: &person {username: Person}\nrepository: {settings: {collaborators: [*person]}}\n"
			}
			if mode == "merged entry" {
				input = "repository: {settings: {collaborators: [{<<: {username: Person}}]}}\n"
			}
			root, path := writebackFixture(t, input)
			record := createdIDRecorder(root, []byte(input))
			expected := input
			if mode == "concurrent edit" {
				expected += "# user edit\n"
				if err := os.WriteFile(path, []byte(expected), 0600); err != nil {
					t.Fatal(err)
				}
			}
			err := record("collaborators", "Person", 3123)
			data, _ := os.ReadFile(path)
			if mode == "success" {
				if err != nil || !strings.Contains(string(data), "id: 3123") || !strings.Contains(string(data), "# keep comment") {
					t.Fatal(string(data), err)
				}
				if err := record("collaborators", "other", 4123); err != nil {
					t.Fatal("second binding lost source snapshot", err)
				}
				info, _ := os.Stat(path)
				if info.Mode().Perm() != 0600 {
					t.Fatal(info.Mode())
				}
			} else if err == nil || string(data) != expected {
				t.Fatal("unsafe binding", string(data), err)
			}
		})
	}
}
