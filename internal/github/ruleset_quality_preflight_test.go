package github

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestManagedQualityUpdateRejectsFreshRawDrift(t *testing.T) {
	for _, mode := range []string{"ok", "changed", "removed", "unknown raw", "denied", "unknown parameter"} {
		t.Run(mode, func(t *testing.T) {
			before, err := decodeRulesetPolicyFromTest([]byte(qualityRulesetJSON))
			if err != nil {
				t.Fatal(err)
			}
			after, err := decodeRulesetPolicyFromTest([]byte(strings.Replace(qualityRulesetJSON, `"errors"`, `"warnings"`, 1)))
			if err != nil {
				t.Fatal(err)
			}
			change, err := settings.NewRulesetChange("update", 7, &before, &after)
			if err != nil {
				t.Fatal(err)
			}
			calls, writes := 0, 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				body := strings.Replace(qualityRulesetJSON, `"errors"`, `"warnings"`, 1)
				if calls == 1 {
					if req.Method != "GET" {
						t.Fatal("missing preflight")
					}
					body = qualityRulesetJSON
					switch mode {
					case "changed":
						body = strings.Replace(body, `"errors"`, `"warnings"`, 1)
					case "removed":
						body = strings.Replace(body, `[{"type":"code_quality","parameters":{"severity":"errors"}}]`, `[]`, 1)
					case "unknown raw":
						body = strings.Replace(body, `"errors"`, `"FUTURE"`, 1)
					case "denied":
						return labelsResponse(req, 403, `{}`), nil
					case "unknown parameter":
						body = strings.Replace(body, `"severity":"errors"`, `"severity":"errors","future":true`, 1)
					}
				} else if req.Method == "PUT" {
					writes++
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || payload["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["severity"] != "warnings" {
						t.Fatal("managed severity payload", err, payload)
					}
				} else if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				return labelsResponse(req, 200, body), nil
			}))
			_, err = client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
			if mode == "ok" {
				if err != nil || writes != 1 || calls != 3 {
					t.Fatal(calls, writes, err)
				}
			} else if err == nil || writes != 0 || calls != 1 {
				t.Fatal("fresh policy overwritten", calls, writes, err)
			}
		})
	}
}
