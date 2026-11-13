package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutolinksLiteralCodec(t *testing.T) {
	const input = "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    autolinks:\n      - id: 7\n        reference_prefix: TICKET-\n        target_url: https://tickets.example.test/<num>?view=full\n        reference_format: numeric\n"
	model, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeConfig(model)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"autolinks:", "reference_format: numeric", "https://tickets.example.test/<num>?view=full"} {
		if !strings.Contains(string(data), want) {
			t.Fatal(string(data))
		}
	}
}

func TestAutolinksStrictInputAndIDRecording(t *testing.T) {
	const good = "{reference_prefix: NEW-, target_url: 'https://target/<num>', reference_format: alphanumeric}"
	for _, item := range []string{"null", "{id: 0, reference_prefix: NEW-, target_url: 'https://x/<num>', reference_format: numeric}", strings.Replace(good, "reference_format: alphanumeric", "reference_format: null", 1), strings.Replace(good, "reference_prefix: NEW-", "key_prefix: NEW-", 1), strings.Replace(good, "reference_format: alphanumeric", "reference_format: false", 1), "{reference_prefix: NEW-, target_url: 'https://x/<num>'}", strings.Replace(good, "reference_format: alphanumeric", "reference_format: numeric, delete: true", 1)} {
		ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {autolinks: [" + item + "]}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted", item)
		}
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	input := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {autolinks: [" + good + "]}}\n# retained\n")
	path := filepath.Join(root, SettingsPath)
	if err := os.WriteFile(path, input, 0600); err != nil {
		t.Fatal(err)
	}
	record := createdIDRecorder(root, input)
	if err := record("autolinks", "NEW-", 8); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	ui, err := DecodeConfig(data)
	if err != nil || *(*ui.Settings.Autolinks)[0].ID != 8 || !strings.Contains(string(data), "# retained") {
		t.Fatal(string(data), err)
	}
	if err = record("autolinks", "NEW-", 9); err == nil {
		t.Fatal("replaced identity")
	}
	fresh := createdIDRecorder(root, input)
	if err = fresh("autolinks", "NEW-", 9); err == nil {
		t.Fatal("stale snapshot replaced")
	}
}
