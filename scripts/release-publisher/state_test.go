package main

import (
	"encoding/json"
	"testing"
)

func TestReleaseStateCheckpoints(t *testing.T) {
	for _, test := range []struct {
		name, checkpoint string
		draft            any
		badMarker        bool
		want             releaseState
		wantError        string
	}{
		{name: "initial draft", checkpoint: "initial", draft: true, want: draftRelease},
		{name: "initial published", checkpoint: "initial", draft: false, want: publishedRelease},
		{name: "legacy draft scalar", checkpoint: "initial", draft: "true", want: draftRelease},
		{name: "legacy published scalar", checkpoint: "initial", draft: "false", want: publishedRelease},
		{name: "legacy trailing newline", checkpoint: "initial", draft: "false\n", want: publishedRelease},
		{name: "initial unknown before marker", checkpoint: "initial", draft: "invalid", badMarker: true, wantError: "unknown release state"},
		{name: "initial marker", checkpoint: "initial", draft: true, badMarker: true, wantError: "release notes do not match tested identity"},
		{name: "observed published does not recheck marker", checkpoint: "observed", draft: false, badMarker: true, want: publishedRelease},
		{name: "observed unknown", checkpoint: "observed", draft: nil, wantError: "unknown release state"},
		{name: "upload draft", checkpoint: "upload", draft: true, want: draftRelease},
		{name: "upload state before marker", checkpoint: "upload", draft: false, badMarker: true, wantError: "release changed state during upload"},
		{name: "upload marker", checkpoint: "upload", draft: true, badMarker: true, wantError: "release notes changed during upload"},
		{name: "publication confirmation", checkpoint: "published", draft: false, badMarker: true, want: publishedRelease},
		{name: "publication not confirmed", checkpoint: "published", draft: true, wantError: "release publication not confirmed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			overrides := map[string]any{"draft": test.draft}
			if test.badMarker {
				overrides["body"] = "wrong"
			}
			var view releaseView
			if err := json.Unmarshal(releaseDocument(t, overrides), &view); err != nil {
				t.Fatal(err)
			}
			got, err := checkedReleaseState(observedIdentity(), view, test.checkpoint)
			message := ""
			if err != nil {
				message = err.Error()
			}
			if got != test.want || message != test.wantError {
				t.Fatalf("state=%q error=%v", got, err)
			}
		})
	}
}

func TestReleaseIDScalarCompatibility(t *testing.T) {
	for _, raw := range []string{`123`, `"123"`, `"123\n"`, `1.23e2`, `123e0`} {
		if got, err := checkedReleaseID(json.RawMessage(raw), "invalid ID"); err != nil || got != "123" {
			t.Fatalf("%s -> %q error=%v", raw, got, err)
		}
	}
	for _, raw := range []string{`0`, `-1`, `null`, `true`, `"00123"`, `1.2`, `123.0`, `1e6`, `1e21`} {
		if got, err := checkedReleaseID(json.RawMessage(raw), "invalid ID"); err == nil || err.Error() != "invalid ID" || got != "" {
			t.Fatalf("%s -> %q error=%v", raw, got, err)
		}
	}
}
