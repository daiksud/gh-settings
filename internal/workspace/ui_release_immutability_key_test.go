package workspace

import (
	"strings"
	"testing"
)

func TestReleaseImmutabilityRejectsObsoleteKey(t *testing.T) {
	for _, fields := range []string{"immutable_release: false", "immutable_release: false, enable_release_immutability: false", "enable_release_immutability: null", "enable_release_immutability: 'true'"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {general: {releases: {" + fields + "}}}}\n")); err == nil {
			t.Fatalf("accepted invalid release control: %s", fields)
		}
	}
	for _, value := range []string{"true", "false"} {
		data := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {general: {releases: {enable_release_immutability: " + value + "}}}}\n")
		model, err := DecodeConfig(data)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeConfig(model)
		if err != nil || !strings.Contains(string(encoded), "enable_release_immutability: "+value) || strings.Contains(string(encoded), "immutable_release:") {
			t.Fatalf("canonical round trip: %s %v", encoded, err)
		}
	}
}
