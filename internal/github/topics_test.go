package github

import (
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func topicsClient(t *testing.T, handle transportFunc) *Client {
	t.Helper()
	rest, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true, Transport: handle})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{rest: rest}
}
func TestReadTopics(t *testing.T) {
	for _, body := range []string{`{"names":["z","a"]}`, `{"names":[]}`, `{}`, `{"names":null}`, `{"names":"a"}`, `{"names":[null]}`, `{"names":["a","a"]}`, `{"names":["Upper"]}`, `not json`} {
		t.Run(body, func(t *testing.T) {
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != "GET" || req.URL.Path != "/repos/owner/current/topics" {
					t.Fatalf("request=%s %s", req.Method, req.URL)
				}
				return &http.Response{Request: req, StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			}))
			got, err := client.ReadTopics(repository.Repository{Owner: "owner", Name: "current"})
			valid := body == `{"names":["z","a"]}` || body == `{"names":[]}`
			if valid && (err != nil || got == nil) || !valid && (err == nil || got != nil) {
				t.Fatalf("names=%v error=%v", got, err)
			}
			if strings.Contains(body, "z") && !reflect.DeepEqual(got, []string{"a", "z"}) {
				t.Fatalf("names=%v", got)
			}
		})
	}
}
func TestApplyTopicsPayloadAndErrors(t *testing.T) {
	for _, status := range []int{200, 403, 422} {
		for _, names := range [][]string{{"z", "a"}, {}} {
			t.Run(strings.Join(names, ",")+http.StatusText(status), func(t *testing.T) {
				calls := 0
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					want := map[string]any{"names": []any{}}
					if len(names) > 0 {
						want["names"] = []any{"a", "z"}
					}
					if req.Method != "PUT" || req.URL.Path != "/repos/owner/current/topics" || !reflect.DeepEqual(payload, want) {
						t.Fatalf("request=%s %s body=%v", req.Method, req.URL, payload)
					}
					return &http.Response{Request: req, StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
				}))
				err := client.ApplyTopics(repository.Repository{Owner: "owner", Name: "current"}, settings.TopicsDeclaration{Names: &names})
				if calls != 1 || (status == 200 && err != nil) || (status != 200 && (err == nil || !strings.Contains(err.Error(), "earlier repository settings update remains applied"))) {
					t.Fatalf("calls=%d error=%v", calls, err)
				}
			})
		}
	}
	client := topicsClient(t, transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("unmanaged request"); return nil, nil }))
	if err := client.ApplyTopics(repository.Repository{}, settings.TopicsDeclaration{}); err != nil {
		t.Fatal(err)
	}
}
