package workspace

import (
	"os"
	"strings"
	"testing"
)

func TestCreatedWebhookIDKeepsControlsAndGuards(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    webhooks:\n      - payload_url: https://exact.example/hook # selected\n        content_type: application/json\n        active: false\n        events: []\n"
	root, path := writebackFixture(t, input)
	record := createdIDRecorder(root, []byte(input))
	if err := record("webhooks", "https://exact.example/hook", 73); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	for _, want := range []string{"id: 73", "# selected", "active: false", "events: []", "content_type: application/json"} {
		if !strings.Contains(string(data), want) {
			t.Fatal("writeback lost control", string(data))
		}
	}
	if _, err := DecodeConfig(data); err != nil {
		t.Fatal(err)
	}
	if err := record("webhooks", "https://exact.example/hook", 74); err == nil {
		t.Fatal("overwrote ID")
	}
	for _, mode := range []string{"edit", "alias", "merge"} {
		t.Run(mode, func(t *testing.T) {
			original := input
			if mode == "alias" {
				original = "version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: [&hook {payload_url: https://exact.example/hook, content_type: application/json, active: false, events: []}, *hook]}}\n"
			}
			if mode == "merge" {
				original = "version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: [{<<: &hook {payload_url: https://exact.example/hook, content_type: application/json, active: false, events: []}}]}}\n"
			}
			r, p := writebackFixture(t, original)
			save := createdIDRecorder(r, []byte(original))
			if mode == "edit" {
				os.WriteFile(p, []byte(original+"# edit\n"), 0600)
			}
			before, _ := os.ReadFile(p)
			if save("webhooks", "https://exact.example/hook", 73) == nil {
				t.Fatal("unsafe writeback accepted")
			}
			after, _ := os.ReadFile(p)
			if string(before) != string(after) {
				t.Fatal("failure changed file")
			}
		})
	}
}
