package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestOIDCImmutableReconcileAndPreserve(t *testing.T) {
	for _, mode := range []string{"default", "custom", "empty", "combined", "matching-true", "matching-false", "unknown", "mode-drift", "order-drift", "immutable-mismatch", "immutable-missing", "prefix-change", "reject-400", "reject-422", "ack-status", "ack-body", "readback-denied", "owner-drift", "archive-drift"} {
		t.Run(mode, func(t *testing.T) {
			yes, no := true, false
			desired := settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}
			state := map[string]any{"use_default": true, "use_immutable_subject": true, "sub_claim_prefix": "old-prefix"}
			if mode == "custom" || mode == "order-drift" || mode == "mode-drift" || mode == "empty" {
				state["use_default"], state["include_claim_keys"] = false, []string{"context", "repo"}
				if mode == "empty" {
					state["include_claim_keys"] = []string{}
				}
			}
			if mode == "combined" {
				keys := []string{"repo", "context"}
				desired.UseDefaultTemplate, desired.SubjectClaimTemplate = &no, &keys
			}
			if mode == "matching-true" {
				desired.UseImmutableSubjectClaim = &yes
			}
			if mode == "matching-false" {
				state["use_immutable_subject"] = false
			}
			if mode == "unknown" {
				delete(state, "use_immutable_subject")
			}
			writes, gets := 0, 0
			c := &OIDCSubjectClient{actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					target := oidcTarget
					if writes > 0 && mode == "owner-drift" {
						target = strings.Replace(target, `"id":2`, `"id":3`, 1)
					}
					if writes > 0 && mode == "archive-drift" {
						target = strings.Replace(target, `"archived":false`, `"archived":true`, 1)
					}
					return aiRepositoryResponse(200, target), nil
				}
				if r.URL.Path != "/repos/owner/current/actions/oidc/customization/sub" {
					t.Fatal(r.URL)
				}
				if r.Method == "GET" {
					gets++
					if writes > 0 && mode == "readback-denied" {
						return aiRepositoryResponse(403, `{"message":"PRIVATE_OIDC"}`), nil
					}
					body, _ := json.Marshal(state)
					return aiRepositoryResponse(200, string(body)), nil
				}
				if r.Method != "PUT" {
					t.Fatal(r.Method)
				}
				writes++
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				want := map[string]any{"use_default": state["use_default"], "use_immutable_subject": false}
				if mode == "combined" {
					want["use_default"], want["include_claim_keys"] = false, []any{"repo", "context"}
				} else if state["use_default"] == false {
					keys := []any{}
					for _, key := range state["include_claim_keys"].([]string) {
						keys = append(keys, key)
					}
					want["include_claim_keys"] = keys
				}
				if !reflect.DeepEqual(payload, want) {
					t.Fatal("fresh companion replay", payload, want)
				}
				if mode == "reject-400" {
					return aiRepositoryResponse(400, `{"message":"PRIVATE_OIDC"}`), nil
				}
				if mode == "reject-422" {
					return aiRepositoryResponse(422, `{"message":"PRIVATE_OIDC"}`), nil
				}
				state = payload
				switch mode {
				case "mode-drift":
					state = map[string]any{"use_default": true, "use_immutable_subject": false}
				case "order-drift":
					state["include_claim_keys"] = []string{"repo", "context"}
				case "immutable-mismatch":
					state["use_immutable_subject"] = true
				case "immutable-missing":
					delete(state, "use_immutable_subject")
				case "prefix-change":
					state["sub_claim_prefix"] = "new-prefix"
				case "ack-status":
					return aiRepositoryResponse(200, `{}`), nil
				case "ack-body":
					return aiRepositoryResponse(201, `{"updated":true}`), nil
				}
				return aiRepositoryResponse(201, `{}`), nil
			}).rest}
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
			matching := strings.HasPrefix(mode, "matching-")
			success := matching || mode == "default" || mode == "custom" || mode == "empty" || mode == "combined" || mode == "prefix-change"
			var pre interface{ ReadFailedBeforeWrite() }
			if (err == nil) != success || wrote != (success && !matching) || errors.As(err, &pre) != (mode == "unknown") {
				t.Fatal(mode, wrote, err)
			}
			wantWrites := 1
			if matching || mode == "unknown" {
				wantWrites = 0
			}
			if writes != wantWrites {
				t.Fatal("unexpected write or retry", writes)
			}
			if success && !matching && gets != 2 {
				t.Fatal("no independent readback", gets)
			}
			if err != nil && strings.Contains(err.Error(), "PRIVATE_OIDC") {
				t.Fatal("upstream error leaked", err)
			}
		})
	}
}

func TestOIDCImmutableOptionalObservation(t *testing.T) {
	for _, leaf := range []string{"", `,"use_immutable_subject":false`, `,"use_immutable_subject":true`, `,"use_immutable_subject":null`, `,"use_immutable_subject":"false"`} {
		c := &OIDCSubjectClient{actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/repos/owner/current" {
				return aiRepositoryResponse(200, oidcTarget), nil
			}
			return aiRepositoryResponse(200, `{"use_default":true`+leaf+`}`), nil
		}).rest}
		o, err := c.Read(repository.Repository{Owner: "owner", Name: "current"})
		bad := strings.Contains(leaf, "null") || strings.Contains(leaf, `"false"`)
		if (err != nil) != bad {
			t.Fatal(leaf, err)
		}
		if bad {
			continue
		}
		v, err := o.Value()
		if err != nil || (v.UseImmutableSubjectClaim == nil) != (leaf == "") {
			t.Fatal(v, err)
		}
		if leaf != "" && *v.UseImmutableSubjectClaim != strings.Contains(leaf, "true") {
			t.Fatal(v)
		}
	}
}
