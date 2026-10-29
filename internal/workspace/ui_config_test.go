package workspace

import (
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIConfigAboutRoundTrip(t *testing.T) {
	want := "version: daiksud/gh-settings@v0\nrepository:\n    about:\n        description: \"\"\n        website: https://example.com\n        topics: []\n"
	got, err := DecodeUIConfig([]byte(want))
	if err != nil {
		t.Fatal(err)
	}
	if got.About == nil || got.About.Description == nil || *got.About.Description != "" || got.About.Website == nil || *got.About.Website != "https://example.com" || got.About.Topics == nil || !reflect.DeepEqual(*got.About.Topics, []string{}) {
		t.Fatalf("lost explicit About intent: %#v", got.About)
	}
	encoded, err := EncodeUIConfig(got)
	if err != nil || string(encoded) != want {
		t.Fatalf("encoded %q, error %v; want %q", encoded, err, want)
	}
}

func TestUIConfigAboutOmission(t *testing.T) {
	got, err := DecodeUIConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {about: {website: ''}}\n"))
	if err != nil || got.About == nil || got.About.Description != nil || got.About.Topics != nil || got.About.Website == nil || *got.About.Website != "" {
		t.Fatalf("omission changed: %#v, %v", got.About, err)
	}
	encoded, err := EncodeUIConfig(uimodel.Repository{})
	if err != nil || string(encoded) != "version: daiksud/gh-settings@v0\nrepository: {}\n" {
		t.Fatalf("empty document: %q, %v", encoded, err)
	}
}

func TestUIConfigRejectsObsoleteEnvelopeAndInvalidAbout(t *testing.T) {
	for _, data := range []string{
		"schemaVersion: 1\nsettings: {}\n",
		"version: daiksud/gh-settings@v0\nsettings: {}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {homepage: x}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {description: null}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {topics: null}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {description: false}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {topics: [true]}}\n",
		"version: daiksud/gh-settings@v0\nrepository: {about: {description: first, description: second}}\n",
		"version: daiksud/gh-settings@v0\nrepository: !!str {}\n",
		"version: daiksud/gh-settings@v0\n---\nversion: daiksud/gh-settings@v0\n",
		"version: 0\n", "version: daiksud/gh-settings@v1\n", "repository: {}\n",
	} {
		t.Run(strings.TrimSpace(data), func(t *testing.T) {
			if _, err := DecodeUIConfig([]byte(data)); err == nil {
				t.Fatal("accepted invalid or obsolete declaration")
			}
		})
	}
}
