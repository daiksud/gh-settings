package settings

import (
	"encoding/json"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestCommitMessageNormalizationJSONRequiresAcquiredStrings(t *testing.T) {
	for _, member := range []string{"path", "raw_title", "raw_message", "fallback"} {
		for _, invalid := range []any{nil, false, float64(1), []any{}, map[string]any{}} {
			fields := map[string]any{"path": "repository.settings.general.pull_requests.squash_merge.default_commit_message", "raw_title": "", "raw_message": "", "fallback": "default_message"}
			fields[member] = invalid
			data, _ := json.Marshal(fields)
			before := CommitMessageNormalization{Path: "unchanged", RawTitle: "unchanged title", RawMessage: "unchanged message", Fallback: uimodel.DefaultMessage}
			got := before
			if err := json.Unmarshal(data, &got); err == nil || got != before {
				t.Fatalf("invalid metadata accepted or partially assigned: %s %+v %v", data, got, err)
			}
		}
		fields := map[string]any{"path": "repository.settings.general.pull_requests.squash_merge.default_commit_message", "raw_title": "", "raw_message": "", "fallback": "default_message"}
		delete(fields, member)
		data, _ := json.Marshal(fields)
		var got CommitMessageNormalization
		if err := json.Unmarshal(data, &got); err == nil {
			t.Fatalf("missing metadata member became acquired empty: %s", data)
		}
	}
	for _, data := range []string{`null`, `[]`, `{"path":"repository.settings.general.pull_requests.squash_merge.default_commit_message","raw_title":"","raw_message":"","fallback":"default_message","unknown_member":"value"}`} {
		var got CommitMessageNormalization
		if err := json.Unmarshal([]byte(data), &got); err == nil {
			t.Fatal("unsupported metadata shape accepted", data)
		}
	}
}

func TestCommitMessageNormalizationJSONRetainsExplicitEmptyPair(t *testing.T) {
	want := CommitMessageNormalization{Path: "repository.settings.general.pull_requests.squash_merge.default_commit_message", RawTitle: "", RawMessage: "", Fallback: uimodel.DefaultMessage}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || len(fields) != 4 || fields["path"] == nil || fields["raw_title"] == nil || fields["raw_message"] == nil || fields["fallback"] == nil {
		t.Fatal("required metadata names or explicit empty members omitted", string(data), err)
	}
	var got CommitMessageNormalization
	if err := json.Unmarshal(data, &got); err != nil || got != want {
		t.Fatal("acquired empty pair lost", got, err)
	}
	if _, err := NewNormalizedCommitMessageChange(SquashCommitMenu, got, uimodel.DefaultMessage); err != nil {
		t.Fatal("valid decoded empty observation rejected", err)
	}
}
