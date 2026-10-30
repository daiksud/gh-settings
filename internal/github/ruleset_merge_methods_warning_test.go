package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestMergeMethodsWarningOwnsListsAndPrecedesWholeWrite(t *testing.T) {
	for _, methods := range [][]string{{"merge", "rebase", "squash"}, {"squash"}, nil} {
		change := mergeMethodsChange(t, methods)
		original := change.Normalizations()
		calls := 0
		client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			if calls != 1 {
				t.Fatal("missing prewrite warning", calls)
			}
			if req.Method == "PUT" {
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(payload, mergeMethodsIntendedPayload(methods)) {
					t.Fatal("sink changed write", payload)
				}
			} else if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			body := mergeMethodsIntendedPayload(methods)
			body["id"], body["source_type"], body["source"] = 7, "Repository", "owner/current"
			encoded, _ := json.Marshal(body)
			return labelsResponse(req, 200, string(encoded)), nil
		}))
		client.RulesetMergeMethodsNormalizationWarning = func(id int64, name string, record settings.RulesetMergeMethodsNormalization) {
			calls++
			if id != 7 || name != "approvals" || record.Path != settings.RulesetMergeMethodsPath || !slices.Equal(record.RawValue, []string{"future", "squash"}) || !slices.Equal(record.Fallback, []string{"merge", "squash", "rebase"}) {
				t.Fatal(id, name, record)
			}
			record.RawValue[0], record.Fallback[0] = "CORRUPT", "CORRUPT"
		}
		id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
		if err != nil || id != 7 || calls != 1 || !reflect.DeepEqual(change.Normalizations(), original) {
			t.Fatal(id, err, calls)
		}
		records := change.Normalizations()
		client.warnRulesetNormalizations(7, "approvals", records)
		if !reflect.DeepEqual(records, original) {
			t.Fatal("callback aliases supplied record")
		}
	}
}
