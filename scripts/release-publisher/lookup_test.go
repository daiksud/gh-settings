package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func observedIdentity() releaseIdentity {
	return releaseIdentity{Version: "v1.2.3", Commit: strings.Repeat("a", 40), Repository: "owner/repo", Marker: "<!-- fixture identity -->"}
}
func releaseDocument(t *testing.T, overrides map[string]any) []byte {
	t.Helper()
	fields := map[string]any{"id": 123, "tag_name": "v1.2.3", "name": "v1.2.3", "prerelease": false, "draft": true, "body": "<!-- fixture identity -->\nreviewed notes\n"}
	for key, value := range overrides {
		fields[key] = value
	}
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestLookupReleaseValidatesIdentityBeforeState(t *testing.T) {
	for _, test := range []struct {
		name, catalog, wantError string
		details                  map[string]any
		exists                   bool
		calls                    int
	}{
		{name: "absent", catalog: `[[]]`, calls: 1},
		{name: "matching draft", exists: true, calls: 2},
		{name: "matching published", details: map[string]any{"draft": false}, exists: true, calls: 2},
		{name: "draft validation deferred", details: map[string]any{"draft": "invalid"}, exists: true, calls: 2},
		{name: "duplicate releases", catalog: `[[{"id":123,"tag_name":"v1.2.3"}],[{"id":124,"tag_name":"v1.2.3"}]]`, calls: 1, wantError: "multiple releases use the requested tag"},
		{name: "invalid listed ID", catalog: `[[{"id":0,"tag_name":"v1.2.3"}]]`, calls: 1, wantError: "release has no valid ID"},
		{name: "changed ID", details: map[string]any{"id": 124, "tag_name": "wrong"}, calls: 2, wantError: "release ID changed"},
		{name: "decimal ID differs", details: map[string]any{"id": json.Number("123.0")}, calls: 2, wantError: "release ID changed"},
		{name: "wrong tag before title", details: map[string]any{"tag_name": "wrong", "name": "wrong"}, calls: 2, wantError: "existing release has another tag"},
		{name: "wrong title before prerelease", details: map[string]any{"name": "wrong", "prerelease": true}, calls: 2, wantError: "existing release has another title"},
		{name: "prerelease before marker", details: map[string]any{"prerelease": true, "body": "wrong"}, calls: 2, wantError: "existing release is a prerelease"},
		{name: "mismatched marker", details: map[string]any{"body": "wrong"}, calls: 2, wantError: "existing release lacks matching version, commit, and binary identity evidence"},
		{name: "marker requires newline", details: map[string]any{"body": "<!-- fixture identity -->"}, calls: 2, wantError: "existing release lacks matching version, commit, and binary identity evidence"},
		{name: "legacy scalar forms", catalog: `[[{"id":"123","tag_name":"v1.2.3"}]]`, details: map[string]any{"id": "123\n", "tag_name": "v1.2.3\n", "name": "v1.2.3\n", "prerelease": "false\n"}, exists: true, calls: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			catalog := test.catalog
			if catalog == "" {
				catalog = `[[{"id":123,"tag_name":"v1.2.3"}]]`
			}
			details := releaseDocument(t, test.details)
			calls := 0
			gh := func(args ...string) ([]byte, error) {
				calls++
				switch calls {
				case 1:
					if !reflect.DeepEqual(args, []string{"api", "repos/owner/repo/releases?per_page=100", "--paginate", "--slurp"}) {
						t.Fatalf("list arguments=%v", args)
					}
					return []byte(catalog), nil
				case 2:
					if !reflect.DeepEqual(args, []string{"api", "repos/owner/repo/releases/123"}) {
						t.Fatalf("detail arguments=%v", args)
					}
					return details, nil
				default:
					t.Fatalf("unexpected API request=%v", args)
					return nil, nil
				}
			}
			work := t.TempDir()
			got, err := lookupRelease(observedIdentity(), work, gh)
			message := ""
			if err != nil {
				message = err.Error()
			}
			want := releaseLookup{Exists: test.exists}
			if test.exists {
				want.ID = "123"
			}
			if message != test.wantError || got != want || calls != test.calls {
				t.Fatalf("lookup=%+v error=%v calls=%d", got, err, calls)
			}
			data, err := os.ReadFile(filepath.Join(work, "releases.json"))
			if err != nil || string(data) != catalog {
				t.Fatalf("raw catalog=%q error=%v", data, err)
			}
			if calls == 2 {
				data, err = os.ReadFile(filepath.Join(work, "release.json"))
				if err != nil || string(data) != string(details) {
					t.Fatalf("raw release=%q error=%v", data, err)
				}
			}
		})
	}
}

func TestLookupAPIFailuresAreNotAbsence(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		for _, status := range []string{"HTTP 404", "HTTP 401"} {
			t.Run(fmt.Sprintf("call%d/%s", failAt, status), func(t *testing.T) {
				cause := &commandError{stderr: []byte("gh: lookup failed (" + status + ")\n"), status: 1}
				calls := 0
				got, err := lookupRelease(observedIdentity(), t.TempDir(), func(args ...string) ([]byte, error) {
					calls++
					if calls == failAt {
						return nil, cause
					}
					return []byte(`[[{"id":123,"tag_name":"v1.2.3"}]]`), nil
				})
				if err != cause || got != (releaseLookup{}) || calls != failAt {
					t.Fatalf("lookup=%+v error=%v calls=%d", got, err, calls)
				}
			})
		}
	}
}

func TestTagIdentityAndLookupBoundary(t *testing.T) {
	sha := strings.Repeat("a", 40)
	other := strings.Repeat("b", 40)
	for _, test := range []struct {
		name, kind, sha, annotatedKind, annotatedSHA, lookupFailure, annotationFailure, wantError string
		optional, exists                                                                          bool
		calls                                                                                     int
	}{
		{name: "lightweight", kind: "commit", sha: sha, optional: true, exists: true, calls: 1},
		{name: "annotated", kind: "tag", sha: other, annotatedKind: "commit", annotatedSHA: sha, optional: true, exists: true, calls: 2},
		{name: "lightweight conflict", kind: "commit", sha: other, optional: true, calls: 1, wantError: "existing tag targets another commit"},
		{name: "annotated conflict", kind: "tag", sha: other, annotatedKind: "commit", annotatedSHA: other, optional: true, calls: 2, wantError: "existing tag targets another commit"},
		{name: "nested annotation rejected", kind: "tag", sha: other, annotatedKind: "tag", annotatedSHA: sha, optional: true, calls: 2, wantError: "existing tag targets another commit"},
		{name: "publication conflict", kind: "commit", sha: other, calls: 1, wantError: "tag changed before publication"},
		{name: "optional 404", lookupFailure: "HTTP 404", optional: true, calls: 1},
		{name: "optional auth failure", lookupFailure: "HTTP 401", optional: true, calls: 1, wantError: "GitHub API lookup failed: repos/owner/repo/git/ref/tags/v1.2.3"},
		{name: "mandatory 404", lookupFailure: "HTTP 404", calls: 1, wantError: "gh: failed (HTTP 404)"},
		{name: "annotation failure", kind: "tag", sha: other, annotationFailure: "HTTP 401", optional: true, calls: 2, wantError: "gh: failed (HTTP 401)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			gh := func(args ...string) ([]byte, error) {
				calls++
				endpoint, kind, target, failure := "repos/owner/repo/git/ref/tags/v1.2.3", test.kind, test.sha, test.lookupFailure
				if calls == 2 {
					endpoint, kind, target, failure = "repos/owner/repo/git/tags/"+other, test.annotatedKind, test.annotatedSHA, test.annotationFailure
				}
				if calls > test.calls || !reflect.DeepEqual(args, []string{"api", endpoint}) {
					t.Fatalf("tag request=%v calls=%d", args, calls)
				}
				if failure != "" {
					return nil, &commandError{stderr: []byte("gh: failed (" + failure + ")\n"), status: 1}
				}
				return []byte(fmt.Sprintf(`{"object":{"type":%q,"sha":%q}}`, kind, target)), nil
			}
			context := "tag changed before publication"
			if test.optional {
				context = "existing tag targets another commit"
			}
			got, err := inspectTag(observedIdentity(), t.TempDir(), test.optional, context, gh)
			message := ""
			if err != nil {
				message = err.Error()
			}
			want := tagObservation{}
			if test.exists {
				want = tagObservation{Exists: true, Type: "commit", SHA: sha}
			}
			if message != test.wantError || got != want || calls != test.calls {
				t.Fatalf("tag=%+v error=%v calls=%d", got, err, calls)
			}
		})
	}
}

func TestTagIsReadAgainAtPublication(t *testing.T) {
	reads := 0
	gh := func(args ...string) ([]byte, error) {
		reads++
		sha := strings.Repeat("a", 40)
		if reads == 2 {
			sha = strings.Repeat("b", 40)
		}
		return []byte(fmt.Sprintf(`{"object":{"type":"commit","sha":%q}}`, sha)), nil
	}
	work := t.TempDir()
	if _, err := inspectTag(observedIdentity(), work, true, "existing tag targets another commit", gh); err != nil {
		t.Fatal(err)
	}
	if got, err := inspectTag(observedIdentity(), work, false, "tag changed before publication", gh); err == nil || err.Error() != "tag changed before publication" || got != (tagObservation{}) || reads != 2 {
		t.Fatalf("tag=%+v error=%v reads=%d", got, err, reads)
	}
}
