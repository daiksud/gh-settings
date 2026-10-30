package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestMergeMethodsFreshWarningsNeverSupplyVerificationValues(t *testing.T) {
	type event struct {
		name string
		raw  []string
	}
	for _, mode := range []string{"canonical", "unknown", "renamed", "malformed", "foreign", "wrong ID", "unrelated", "denied"} {
		t.Run(mode, func(t *testing.T) {
			change := mergeMethodsChange(t, []string{"merge", "rebase", "squash"})
			var got []event
			var operations []string
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				operations = append(operations, req.Method)
				body := mergeMethodsReadBody([]string{"merge", "rebase", "squash"}, false)
				if req.Method == "GET" {
					if mode == "denied" {
						return labelsResponse(req, 403, `{"message":"denied"}`), nil
					}
					if mode != "canonical" {
						body = mergeMethodsReadBody([]string{"squash", "FRESH_METHOD"}, false)
					}
					switch mode {
					case "renamed":
						body["name"] = "fresh name"
					case "malformed":
						body["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["allowed_merge_methods"] = []any{nil}
					case "foreign":
						body["source"] = "owner/other"
					case "wrong ID":
						body["id"] = 8
					case "unrelated":
						body["future"] = true
					}
				} else if req.Method != "PUT" {
					t.Fatal(req.Method)
				}
				encoded, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(encoded)), nil
			}))
			client.RulesetMergeMethodsNormalizationWarning = func(id int64, name string, n settings.RulesetMergeMethodsNormalization) {
				if id != 7 || n.Path != settings.RulesetMergeMethodsPath || !slices.Equal(n.Fallback, []string{"merge", "squash", "rebase"}) {
					t.Fatal(id, name, n)
				}
				got = append(got, event{name: name, raw: slices.Clone(n.RawValue)})
				n.RawValue[0], n.Fallback[0] = "CORRUPT", "CORRUPT"
			}
			id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
			want := []event{{name: "approvals", raw: []string{"future", "squash"}}}
			if mode == "unknown" || mode == "renamed" {
				name := "approvals"
				if mode == "renamed" {
					name = "fresh name"
				}
				want = append(want, event{name: name, raw: []string{"squash", "FRESH_METHOD"}})
			}
			if id != 7 || (err != nil) != (mode != "canonical") || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(operations, []string{"PUT", "GET"}) {
				t.Fatal(id, err, got, want, operations)
			}
			if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatal(err)
			}
		})
	}
}
