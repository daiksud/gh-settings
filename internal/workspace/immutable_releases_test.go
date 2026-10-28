package workspace

import (
	"strings"
	"testing"
)

func TestImmutableReleasesYAML(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {security: {immutable_releases: " + value + "}}"))
		if err != nil {
			t.Fatal(err)
		}
		if d.Empty() || !d.Security.SecretScanningEmpty() || d.Security.ImmutableReleases == nil {
			t.Fatal(d)
		}
		data, err := EncodeConfig(d)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeConfig(data)
		if err != nil || again.Security.ImmutableReleases == nil || *again.Security.ImmutableReleases != (value == "true") || strings.Contains(string(data), "secret_scanning:") {
			t.Fatal(string(data), again, err)
		}
	}
	for _, value := range []string{"null", "'true'", "[]", "{enabled: true}", "yes", "1"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {security: {immutable_releases: " + value + "}}")); err == nil {
			t.Fatal("invalid value accepted", value)
		}
	}
	for _, input := range []string{"schemaVersion: 1\nsettings: {security: {immutable_releases: true, immutable_releases: false}}", "schemaVersion: 1\nsettings: {security: {immutable_release: true}}"} {
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatal("invalid key accepted", input)
		}
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {security: {secret_scanning: false, immutable_releases: true}}"))
	if err != nil || d.Security.SecretScanningEmpty() || d.Security.ImmutableReleases == nil {
		t.Fatal("independent fields coupled", d, err)
	}
}
