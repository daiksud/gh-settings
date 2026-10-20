package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPublishStopsAfterFirstUploadError(t *testing.T) {
	version, env := preflightFixture(t)
	identity, err := prepareIdentity(version, func(key string) string { return env[key] })
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("first upload failed")
	wantCalls := [][]string{
		{"api", "repos/owner/repo/releases?per_page=100", "--paginate", "--slurp"},
		{"api", "repos/owner/repo/releases/123"},
		{"api", "repos/owner/repo/git/ref/tags/v1.2.3"},
		{"api", "repos/owner/repo/releases/123"},
		{"release", "upload", "v1.2.3", env["RELEASE_DIR"] + "/gh-settings-darwin-arm64", "--repo", "owner/repo"},
	}
	var calls [][]string
	gh := func(args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		switch {
		case reflect.DeepEqual(args, wantCalls[0]):
			return []byte(`[[{"id":123,"tag_name":"v1.2.3"}]]`), nil
		case reflect.DeepEqual(args, wantCalls[1]):
			// A matching draft with all three assets missing requires uploads.
			return releaseDocument(t, map[string]any{
				"body": identity.Marker + "\nreviewed notes\n", "assets": []any{},
			}), nil
		case reflect.DeepEqual(args, wantCalls[2]):
			return []byte(fmt.Sprintf(`{"object":{"type":"commit","sha":%q}}`, identity.Commit)), nil
		case len(args) >= 2 && args[0] == "release" && args[1] == "upload":
			return nil, cause
		default:
			return nil, fmt.Errorf("unexpected GitHub operation: %v", args)
		}
	}
	var output bytes.Buffer
	err = publishRelease(identity, "/unavailable-history", t.TempDir(), &output, gh)
	// The complete trace rejects later uploads, reads, tag checks, or publication,
	// even if the original error is eventually returned after that extra work.
	if !errors.Is(err, cause) || !reflect.DeepEqual(calls, wantCalls) || output.Len() != 0 {
		t.Fatalf("error=%v calls=%v output=%q, want original error and calls=%v", err, calls, output.String(), wantCalls)
	}
}

func TestPublishDraftChecksFreshAssetsStateMarkerThenTag(t *testing.T) {
	for _, test := range []struct {
		name, asset, marker, tag, confirmation, failure, want string
		draft                                                 any
		initialPublished                                      bool
		tagReads, publishes, reads                            int
	}{
		{name: "complete", tagReads: 2, publishes: 1, reads: 4},
		{name: "asset before state marker and tag", asset: "unexpected", draft: false, marker: "changed", tag: "changed", want: "unexpected release asset", tagReads: 1, reads: 3},
		{name: "state before marker and tag", draft: false, marker: "changed", tag: "changed", want: "release changed state during upload", tagReads: 1, reads: 3},
		{name: "unknown state before marker", draft: "unknown", marker: "changed", want: "release changed state during upload", tagReads: 1, reads: 3},
		{name: "marker before tag", marker: "changed", tag: "changed", want: "release notes changed during upload", tagReads: 1, reads: 3},
		{name: "fresh tag", tag: "changed", want: "tag changed before publication", tagReads: 2, reads: 3},
		{name: "publication failure", failure: "PATCH", want: "transport failed", tagReads: 2, publishes: 1, reads: 3},
		{name: "confirmation failure", failure: "confirmation", want: "transport failed", tagReads: 2, publishes: 1, reads: 4},
		{name: "unconfirmed publication", confirmation: "true", want: "release publication not confirmed", tagReads: 2, publishes: 1, reads: 4},
		{name: "confirmation cannot reuse initial state", initialPublished: true, confirmation: "missing", want: "release publication not confirmed", tagReads: 2, publishes: 1, reads: 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			version, env := preflightFixture(t)
			identity, err := prepareIdentity(version, func(key string) string { return env[key] })
			if err != nil {
				t.Fatal(err)
			}
			reads, tags, publishes := 0, 0, 0
			var downloads []string
			gh := func(args ...string) ([]byte, error) {
				switch {
				case reflect.DeepEqual(args, []string{"api", "repos/owner/repo/releases?per_page=100", "--paginate", "--slurp"}):
					return []byte(`[[{"id":123,"tag_name":"v1.2.3"}]]`), nil
				case reflect.DeepEqual(args, []string{"api", "repos/owner/repo/releases/123"}):
					reads++
					fields := map[string]any{"id": 123, "tag_name": identity.Version, "name": identity.Version, "prerelease": false, "draft": true, "body": identity.Marker + "\nreviewed notes\n", "assets": []map[string]string{{"name": macosAsset}, {"name": linuxAsset}, {"name": checksumsAsset}}}
					if reads == 1 && test.initialPublished {
						fields["draft"] = false
					}
					if reads == 3 {
						if test.asset != "" {
							fields["assets"] = []map[string]string{{"name": test.asset}}
						}
						if test.draft != nil {
							fields["draft"] = test.draft
						}
						if test.marker != "" {
							fields["body"] = test.marker
						}
					}
					if reads == 4 {
						if test.failure == "confirmation" {
							return nil, fmt.Errorf("transport failed")
						}
						fields["draft"] = test.confirmation == "true"
						if test.confirmation == "missing" {
							delete(fields, "draft")
						}
					}
					return json.Marshal(fields)
				case reflect.DeepEqual(args, []string{"api", "repos/owner/repo/git/ref/tags/v1.2.3"}):
					tags++
					sha := identity.Commit
					if tags == 2 && test.tag != "" {
						sha = strings.Repeat("b", 40)
					}
					return []byte(fmt.Sprintf(`{"object":{"type":"commit","sha":%q}}`, sha)), nil
				case len(args) == 9 && reflect.DeepEqual(args[:6], []string{"release", "download", identity.Version, "--repo", identity.Repository, "--pattern"}) && args[7] == "--dir":
					name := args[6]
					downloads = append(downloads, name)
					data, err := os.ReadFile(identity.Directory + "/" + name)
					if err != nil {
						t.Fatal(err)
					}
					return nil, os.WriteFile(filepath.Join(args[8], name), data, 0600)
				case len(args) == 6 && reflect.DeepEqual(args[:5], []string{"api", "-X", "PATCH", "repos/owner/repo/releases/123", "--input"}):
					publishes++
					data, err := os.ReadFile(args[5])
					if err != nil || string(data) != "{\"draft\":false}\n" || reads != 3 || tags != 2 {
						t.Fatalf("publication input=%q error=%v reads=%d tags=%d", data, err, reads, tags)
					}
					if test.failure == "PATCH" {
						return nil, fmt.Errorf("transport failed")
					}
					return nil, nil
				default:
					t.Fatalf("unexpected read or mutation: %v", args)
					return nil, nil
				}
			}
			var output bytes.Buffer
			// This nonexistent root proves an existing draft never regenerates notes.
			err = publishRelease(identity, "/unavailable-history", t.TempDir(), &output, gh)
			message := ""
			if err != nil {
				message = err.Error()
			}
			if message != test.want || reads != test.reads || tags != test.tagReads || publishes != test.publishes {
				t.Fatalf("error=%v reads=%d tags=%d publishes=%d", err, reads, tags, publishes)
			}
			wantDownloads := []string{macosAsset, linuxAsset, checksumsAsset}
			if test.asset == "" {
				wantDownloads = append(wantDownloads, macosAsset, linuxAsset, checksumsAsset)
			}
			if !reflect.DeepEqual(downloads, wantDownloads) {
				t.Fatalf("downloads=%v want=%v", downloads, wantDownloads)
			}
			wantOutput := ""
			if test.want == "" {
				wantOutput = "published v1.2.3 at " + identity.Commit + " with verified assets\n"
			}
			if output.String() != wantOutput {
				t.Fatalf("output=%q want=%q", output.String(), wantOutput)
			}
		})
	}
}
