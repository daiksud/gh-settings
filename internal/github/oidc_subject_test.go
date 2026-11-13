package github

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

const oidcTarget = `{"id":1,"full_name":"owner/current","visibility":"public","archived":false,"owner":{"id":2,"type":"User"}}`

func oidcDesired(defaults bool, keys ...string) settings.OIDCSubjectTemplate {
	v := settings.OIDCSubjectTemplate{UseDefaultTemplate: &defaults}
	if !defaults {
		if keys == nil {
			keys = []string{}
		}
		v.SubjectClaimTemplate = &keys
	}
	return v
}
func TestOIDCObservationCompleteness(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `{"use_default":null}`, `{"use_default":"true"}`, `{"use_default":false}`, `{"use_default":false,"include_claim_keys":null}`, `{"use_default":false,"include_claim_keys":[null]}`, `{"use_default":false,"include_claim_keys":["repo","repo"]}`, `{"use_default":true,"use_immutable_subject":null}`, `{"use_default":true,"sub_claim_prefix":false}`, `{"use_default":true,"future":true}`, `{"use_default":true} {}`} {
		c := &OIDCSubjectClient{actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == "/repos/owner/current" {
				return aiRepositoryResponse(200, oidcTarget), nil
			}
			return aiRepositoryResponse(200, body), nil
		}).rest}
		if _, err := c.Read(repository.Repository{Owner: "owner", Name: "current"}); err == nil {
			t.Fatal("accepted", body)
		}
	}
	for _, body := range []string{`{"use_default":true}`, `{"use_default":false,"include_claim_keys":[]}`, `{"use_default":true,"include_claim_keys":["repo"],"use_immutable_subject":false}`} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(body), &raw)
		v, _, err := decodeOIDCSubject(raw)
		if err != nil {
			t.Fatal(body, err)
		}
		if *v.UseDefaultTemplate && v.SubjectClaimTemplate != nil {
			t.Fatal("inactive keys exported")
		}
	}
	for _, target := range []string{`{}`, strings.Replace(oidcTarget, `"User"`, `"Organization"`, 1), strings.Replace(oidcTarget, `"public"`, `"private"`, 1), strings.Replace(oidcTarget, `owner/current`, `other/current`, 1), strings.Replace(oidcTarget, `"archived":false,`, ``, 1)} {
		calls := 0
		c := &OIDCSubjectClient{actionsClient(t, func(*http.Request) (*http.Response, error) { calls++; return aiRepositoryResponse(200, target), nil }).rest}
		if _, err := c.Read(repository.Repository{Owner: "owner", Name: "current"}); err == nil || calls != 1 {
			t.Fatal(target, err, calls)
		}
	}
}
func TestOIDCUpdateFreshNoOpReplayAndVerification(t *testing.T) {
	for _, mode := range []string{"custom", "default", "empty", "known false", "no-op unknown", "unknown write", "archived", "read denied", "write denied", "write network", "wrong status", "ack data", "ack trailing", "ack null", "ack object", "readback denied", "readback mode", "readback order", "readback immutable", "readback unknown", "target drift", "owner drift", "archive drift"} {
		t.Run(mode, func(t *testing.T) {
			d := oidcDesired(false, "repo", "context")
			state := `{"use_default":true,"use_immutable_subject":true}`
			if mode == "default" {
				d = oidcDesired(true)
				state = `{"use_default":false,"include_claim_keys":["repo"],"use_immutable_subject":true}`
			}
			if mode == "empty" {
				d = oidcDesired(false)
			}
			if mode == "known false" {
				state = `{"use_default":true,"use_immutable_subject":false}`
			}
			if mode == "no-op unknown" {
				state = `{"use_default":false,"include_claim_keys":["repo","context"]}`
			}
			if mode == "unknown write" {
				state = `{"use_default":true}`
			}
			writes, reads := 0, 0
			c := &OIDCSubjectClient{actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					target := oidcTarget
					if mode == "archived" || writes > 0 && mode == "archive drift" {
						target = strings.Replace(target, `"archived":false`, `"archived":true`, 1)
					}
					if writes > 0 && mode == "target drift" {
						target = strings.Replace(target, `"id":1`, `"id":3`, 1)
					}
					if writes > 0 && mode == "owner drift" {
						target = strings.Replace(target, `"id":2`, `"id":3`, 1)
					}
					return aiRepositoryResponse(200, target), nil
				}
				if r.URL.Path != "/repos/owner/current/actions/oidc/customization/sub" {
					t.Fatal(r.URL)
				}
				if r.Method == "GET" {
					reads++
					if mode == "read denied" || writes > 0 && mode == "readback denied" {
						return aiRepositoryResponse(403, `{"message":"private-canary"}`), nil
					}
					return aiRepositoryResponse(200, state), nil
				}
				if r.Method != "PUT" {
					t.Fatal(r.Method)
				}
				writes++
				body, _ := io.ReadAll(r.Body)
				var input map[string]any
				if json.Unmarshal(body, &input) != nil {
					t.Fatal(string(body))
				}
				want := 2
				if !*d.UseDefaultTemplate {
					want = 3
				}
				if len(input) != want || input["use_default"] != *d.UseDefaultTemplate || input["use_immutable_subject"] != (mode != "known false") {
					t.Fatal("payload", string(body))
				}
				if !*d.UseDefaultTemplate {
					wantKeys := `["repo","context"]`
					if mode == "empty" {
						wantKeys = `[]`
					}
					keys, _ := json.Marshal(input["include_claim_keys"])
					if string(keys) != wantKeys {
						t.Fatal("ordered payload keys", string(keys))
					}
				}
				if mode == "write network" {
					return nil, errors.New("private-canary")
				}
				if mode == "write denied" {
					return aiRepositoryResponse(403, `{"message":"private-canary"}`), nil
				}
				b, _ := json.Marshal(input)
				state = string(b)
				switch mode {
				case "readback mode":
					state = `{"use_default":true,"use_immutable_subject":true}`
				case "readback order":
					state = `{"use_default":false,"include_claim_keys":["context","repo"],"use_immutable_subject":true}`
				case "readback immutable":
					state = strings.Replace(state, `"use_immutable_subject":true`, `"use_immutable_subject":false`, 1)
				case "readback unknown":
					state = `{"use_default":false,"include_claim_keys":["repo","context"]}`
				}
				status, ack := 201, ""
				switch mode {
				case "wrong status":
					status = 200
				case "ack data":
					ack = `{"unexpected":true}`
				case "ack trailing":
					ack = `{} {}`
				case "ack null":
					ack = `null`
				case "ack object":
					ack = `{ }`
				}
				return aiRepositoryResponse(status, ack), nil
			}).rest}
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, d)
			success := mode == "custom" || mode == "default" || mode == "empty" || mode == "known false" || mode == "no-op unknown" || mode == "ack null" || mode == "ack object"
			pre := mode == "unknown write" || mode == "archived" || mode == "read denied"
			var marker interface{ ReadFailedBeforeWrite() }
			if (err == nil) != success || wrote != (success && mode != "no-op unknown") || errors.As(err, &marker) != pre {
				t.Fatal(wrote, err, pre)
			}
			if (pre || mode == "no-op unknown") && writes != 0 {
				t.Fatal("unexpected write")
			}
			if success && wrote && (writes != 1 || reads != 2) {
				t.Fatal("missing independent read", writes, reads)
			}
			if err != nil && strings.Contains(err.Error(), "private-canary") {
				t.Fatal("raw error leaked")
			}
		})
	}
}
func TestOIDCAcknowledgmentShapes(t *testing.T) {
	for _, body := range []string{"", " \n", "null", "{}", " { }\n"} {
		if !validOIDCAcknowledgment([]byte(body)) {
			t.Fatal(body)
		}
	}
	for _, body := range []string{`[]`, `false`, `0`, `""`, `{"x":null}`, `{}null`, `null{}`} {
		if validOIDCAcknowledgment([]byte(body)) {
			t.Fatal(body)
		}
	}
}

func TestOIDCBoundedResponses(t *testing.T) {
	if _, err := readOIDCBody(strings.NewReader(strings.Repeat(" ", oidcResponseLimit+1))); err == nil {
		t.Fatal("accepted oversized response")
	}
	if _, err := readOIDCBody(strings.NewReader(strings.Repeat(" ", oidcResponseLimit))); err != nil {
		t.Fatal(err)
	}
}
