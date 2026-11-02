package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"reflect"
	"testing"
)

const pagesFixture = `{"build_type":"legacy","source":{"branch":"main","path":"/docs"},"cname":"example.com","https_enforced":true,"public":true}`

func TestPagesReadRejectsUnknownAndIncomplete(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, key := range []string{"build_type", "source", "cname", "https_enforced", "public"} {
		for _, missing := range []bool{true, false} {
			var raw map[string]any
			json.Unmarshal([]byte(pagesFixture), &raw)
			if missing {
				delete(raw, key)
			} else {
				raw[key] = nil
			}
			body, _ := json.Marshal(raw)
			c := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" || r.URL.Path != "/repos/owner/current/pages" {
					t.Fatal(r.Method, r.URL)
				}
				return actionsResponse(r, 200, string(body)), nil
			}).rest}
			o, err := c.Read(repo)
			valid := key == "cname" && !missing
			if (err == nil) != valid {
				t.Fatal(key, missing, err)
			}
			if !valid {
				if _, err = o.Values(); err == nil {
					t.Fatal("failed read acquired")
				}
			}
		}
	}
	for _, status := range []int{301, 302, 307, 401, 403, 500} {
		c := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			resp := actionsResponse(r, status, `{}`)
			resp.Header.Set("Location", "https://other.invalid/pages")
			return resp, nil
		}).rest}
		if _, err := c.Read(repo); err == nil {
			t.Fatal("unavailable accepted", status)
		}
	}
}
func TestPagesNarrowUpdateReadbackAndFailureCertainty(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, failure := range []string{"", "404", "identity", "write", "readback", "companion", "visibility", "ignored", "redirect", "archived"} {
		t.Run(failure, func(t *testing.T) {
			writes := 0
			identityReads := 0
			var state map[string]any
			json.Unmarshal([]byte(pagesFixture), &state)
			c := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					if r.Method != "GET" {
						t.Fatal("identity mutation")
					}
					identityReads++
					id := 123
					if failure == "identity" && identityReads > 1 {
						id = 456
					}
					return actionsResponse(r, 200, fmt.Sprintf(`{"id":%d,"full_name":"owner/current","archived":%t}`, id, failure == "archived")), nil
				}
				if r.URL.Path != "/repos/owner/current/pages" {
					t.Fatal("unrelated endpoint", r.URL)
				}
				if r.Method == "GET" {
					if failure == "404" || failure == "readback" && writes > 0 {
						return actionsResponse(r, 404, `{}`), nil
					}
					body, _ := json.Marshal(state)
					return actionsResponse(r, 200, string(body)), nil
				}
				if r.Method != "PUT" {
					t.Fatal("lifecycle/build mutation", r.Method)
				}
				writes++
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if !reflect.DeepEqual(body, map[string]any{"cname": nil}) {
					t.Fatal("unowned payload", body)
				}
				if failure == "write" {
					return actionsResponse(r, 422, `{}`), nil
				}
				if failure == "redirect" {
					response := actionsResponse(r, 307, `{}`)
					response.Header.Set("Location", "https://other.invalid")
					return response, nil
				}
				if failure != "ignored" {
					state["cname"] = nil
				}
				if failure == "companion" {
					state["https_enforced"] = false
				}
				if failure == "visibility" {
					state["public"] = false
				}
				return actionsResponse(r, 204, ""), nil
			}).rest}
			wrote, err := c.Update(repo, settings.PagesDeclaration{CNAME: &settings.PagesDomain{}})
			if (err == nil) != (failure == "") || wrote != (failure == "") {
				t.Fatal(wrote, err)
			}
			var pre interface{ ReadFailedBeforeWrite() }
			wantPre := failure == "404" || failure == "identity" || failure == "archived"
			if errors.As(err, &pre) != wantPre {
				t.Fatal("wrong certainty", err)
			}
			want := 1
			if wantPre {
				want = 0
			}
			if writes != want {
				t.Fatal("unexpected mutation count", writes)
			}
			if failure == "" {
				wrote, err = c.Update(repo, settings.PagesDeclaration{CNAME: &settings.PagesDomain{}})
				if wrote || err != nil || writes != 1 {
					t.Fatal("matching write", wrote, err, writes)
				}
			}
		})
	}
}
