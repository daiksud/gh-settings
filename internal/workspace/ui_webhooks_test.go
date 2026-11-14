package workspace

import (
	"strings"
	"testing"
)

func TestUIWebhooksLiteralRoundtrip(t *testing.T) {
	input := "version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    webhooks:\n      - id: 123\n        payload_url: https://EXAMPLE.com:8443/github?key=%2f\n        content_type: application/json\n      - id: 124\n        content_type: application/x-www-form-urlencoded\n"
	value, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeConfig(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"id: 123", "https://EXAMPLE.com:8443/github?key=%2f", "content_type: application/json", "content_type: application/x-www-form-urlencoded"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("roundtrip missing %q", want)
		}
	}
}

func TestUIWebhooksRejectInvalidShapes(t *testing.T) {
	for _, value := range []string{"null", "[{id: null, content_type: application/json}]", "[{id: '7', content_type: application/json}]", "[{id: 7, content_type: null}]", "[{id: 7, Content_type: application/json}]", "[{id: 7, content_type: application/json, content_type: application/json}]", "[{id: 7, payload_url: false}]", "[{id: 7, url: https://example.com}]", "[{id: 7, secret: PRIVATE}]", "[{id: 7, insecure_ssl: false}]"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository:\n  settings:\n    webhooks: " + value + "\n")); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}
