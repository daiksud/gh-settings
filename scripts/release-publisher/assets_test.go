package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestVerifyAssetsDownloadsBytesAndReportsMissing(t *testing.T) {
	for _, test := range []struct {
		name                         string
		names                        []any
		failure, tamper, omit        string
		apiFailure                   bool
		verified, missing, downloads []string
		wantError                    string
	}{
		{name: "complete", verified: []string{"gh-settings-darwin-arm64", "gh-settings-linux-amd64", "SHA256SUMS"}, missing: []string{}, downloads: []string{"gh-settings-darwin-arm64", "gh-settings-linux-amd64", "SHA256SUMS"}},
		{name: "missing checksum", names: []any{"gh-settings-linux-amd64", "gh-settings-darwin-arm64"}, verified: []string{"gh-settings-darwin-arm64", "gh-settings-linux-amd64"}, missing: []string{"SHA256SUMS"}, downloads: []string{"gh-settings-darwin-arm64", "gh-settings-linux-amd64"}},
		{name: "missing binaries", names: []any{"SHA256SUMS"}, verified: []string{"SHA256SUMS"}, missing: []string{"gh-settings-darwin-arm64", "gh-settings-linux-amd64"}, downloads: []string{"SHA256SUMS"}},
		{name: "empty", names: []any{}, verified: []string{}, missing: []string{"gh-settings-darwin-arm64", "gh-settings-linux-amd64", "SHA256SUMS"}},
		{name: "duplicate", names: []any{"gh-settings-linux-amd64", "gh-settings-linux-amd64"}, wantError: "duplicate asset names"},
		{name: "unexpected", names: []any{"other"}, wantError: "unexpected release asset"},
		{name: "duplicate before unexpected", names: []any{"other", "other"}, wantError: "duplicate asset names"},
		{name: "names are not trimmed", names: []any{"gh-settings-linux-amd64\n"}, wantError: "unexpected release asset"},
		{name: "API failure", apiFailure: true, wantError: "gh: read failed (HTTP 401)"},
		{name: "first download fails", failure: "gh-settings-darwin-arm64", downloads: []string{"gh-settings-darwin-arm64"}, wantError: "gh: download failed"},
		{name: "later download fails", failure: "gh-settings-linux-amd64", downloads: []string{"gh-settings-darwin-arm64", "gh-settings-linux-amd64"}, wantError: "gh: download failed"},
		{name: "downloaded bytes mismatch", tamper: "gh-settings-linux-amd64", downloads: []string{"gh-settings-darwin-arm64", "gh-settings-linux-amd64"}, wantError: "artifact hash mismatch: gh-settings-linux-amd64"},
		{name: "download reported success without bytes", omit: "gh-settings-linux-amd64", downloads: []string{"gh-settings-darwin-arm64", "gh-settings-linux-amd64"}, wantError: "artifact hash mismatch: gh-settings-linux-amd64"},
	} {
		t.Run(test.name, func(t *testing.T) {
			version, env := preflightFixture(t)
			identity, err := prepareIdentity(version, func(key string) string { return env[key] })
			if err != nil {
				t.Fatal(err)
			}
			names := test.names
			if names == nil {
				names = []any{"gh-settings-linux-amd64", "SHA256SUMS", "gh-settings-darwin-arm64"}
			}
			assets := make([]map[string]any, 0, len(names))
			for _, name := range names {
				assets = append(assets, map[string]any{"name": name, "digest": "untrusted metadata"})
			}
			// State and marker checks belong after byte verification, not here.
			body, err := json.Marshal(map[string]any{"assets": assets, "draft": "unknown", "body": "changed marker"})
			if err != nil {
				t.Fatal(err)
			}
			work := t.TempDir()
			var downloads []string
			reads := 0
			gh := func(args ...string) ([]byte, error) {
				if len(args) == 2 && args[0] == "api" {
					reads++
					if args[1] != "repos/owner/repo/releases/123" || reads != 1 {
						t.Fatalf("refresh request=%v", args)
					}
					if test.apiFailure {
						return nil, &commandError{stderr: []byte("gh: read failed (HTTP 401)\n"), status: 1}
					}
					return body, nil
				}
				if reads != 1 || len(args) != 9 || !reflect.DeepEqual(args[:6], []string{"release", "download", "v1.2.3", "--repo", "owner/repo", "--pattern"}) || args[7] != "--dir" || filepath.Dir(args[8]) != work || !strings.HasPrefix(filepath.Base(args[8]), "downloads.") {
					t.Fatalf("unexpected download/mutation=%v", args)
				}
				name := args[6]
				downloads = append(downloads, name)
				if name == test.failure {
					return nil, &commandError{stderr: []byte("gh: download failed\n"), status: 1}
				}
				if name == test.omit {
					return nil, nil
				}
				bytes, ok := map[string]string{"gh-settings-darwin-arm64": "macos-binary", "gh-settings-linux-amd64": "linux-binary", "SHA256SUMS": testChecksums}[name]
				if !ok {
					t.Fatalf("unexpected asset %q", name)
				}
				if name == test.tamper {
					bytes = "changed bytes"
				}
				if err := os.WriteFile(filepath.Join(args[8], name), []byte(bytes), 0600); err != nil {
					t.Fatal(err)
				}
				return nil, nil
			}
			got, err := verifyAssets(identity, "123", work, gh)
			message := ""
			if err != nil {
				message = err.Error()
			}
			want := assetVerification{Verified: test.verified, Missing: test.missing}
			if test.wantError == "" {
				if err := json.Unmarshal(body, &want.view); err != nil {
					t.Fatal(err)
				}
			}
			if message != test.wantError || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(downloads, test.downloads) || reads != 1 {
				t.Fatalf("assets=%+v error=%v downloads=%v reads=%d", got, err, downloads, reads)
			}
		})
	}
}

func TestVerifyAssetsRequiresObservedCollection(t *testing.T) {
	for _, body := range []string{`{}`, `{"assets":null}`} {
		got, err := verifyAssets(observedIdentity(), "123", t.TempDir(), func(args ...string) ([]byte, error) { return []byte(body), nil })
		if err == nil || err.Error() != "missing release assets" || !reflect.DeepEqual(got, assetVerification{}) {
			t.Fatalf("body=%s assets=%+v error=%v", body, got, err)
		}
	}
}
