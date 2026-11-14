package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"strings"
	"testing"
)

func TestUIWebhookCreationLiteralRoundtrip(t *testing.T) {
	for _, entry := range []string{
		"{payload_url: https://example.com/github, content_type: application/json, active: false, events: []}",
		"{id: 73, active: false, events: [push, future_event]}",
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: [" + entry + "]}}\n"
		value, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = github.DeclarationFromUI(value); err != nil {
			t.Fatal(err)
		}
		data, err := EncodeConfig(value)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "active: false") || !strings.Contains(string(data), "events:") {
			t.Fatalf("lost controls: %s", data)
		}
		if _, err = DecodeConfig(data); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUIWebhookCreationRejectsIncompleteAndInvalidControls(t *testing.T) {
	for _, entry := range []string{
		"{payload_url: https://example.com, content_type: application/json, events: []}",
		"{payload_url: https://example.com, content_type: application/json, active: false}",
		"{id: 0, payload_url: https://example.com, content_type: application/json, active: false, events: []}",
		"{id: 7, active: 'false'}", "{id: 7, active: null}", "{id: 7, events: null}",
		"{id: 7, events: [push, push]}", "{id: 7, events: [1]}", "{id: 7, events: ['bad event']}",
		"{id: 7, events: [\"bad\\ncontrol\"]}", "{id: 7, events: [] , events: []}",
	} {
		value, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: [" + entry + "]}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(value)
		}
		if err == nil {
			t.Fatal("invalid declaration accepted", entry)
		}
	}
}

func TestUIWebhookSSLVerificationRoundtrip(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: [{id: 7, ssl_verification: " + value + "}]}}\n"
		ui, err := DecodeConfig([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		d, err := github.DeclarationFromUI(ui)
		if err != nil || d.Webhooks == nil || (*d.Webhooks)[0].SSLVerification == nil || *(*d.Webhooks)[0].SSLVerification != (value == "true") {
			t.Fatal(d, err)
		}
		output, err := EncodeConfig(ui)
		if err != nil || !strings.Contains(string(output), "ssl_verification: "+value) {
			t.Fatal(string(output), err)
		}
	}
	for _, value := range []string{"null", "'false'", "0"} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {webhooks: [{id: 7, ssl_verification: " + value + "}]}}\n"
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatal("accepted", value)
		}
	}
}
