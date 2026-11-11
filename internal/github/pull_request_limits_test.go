package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

const capBaseline = `{"enabled":false,"max_open_pull_requests":7,"include_drafts":true}`

func TestPullRequestLimitsRead(t *testing.T) {
	for _, body := range []string{capBaseline, `{"enabled":false,"max_open_pull_requests":7}`, `{"enabled":false,"max_open_pull_requests":7,"include_drafts":null}`, `{"enabled":"false","max_open_pull_requests":7,"include_drafts":true}`, `{"enabled":false,"max_open_pull_requests":0,"include_drafts":true}`, `{"enabled":false,"max_open_pull_requests":1001,"include_drafts":true}`, capBaseline + ` {}`} {
		t.Run(body, func(t *testing.T) {
			c := &PullRequestLimitsClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, body), nil })).rest}
			o, err := c.Read(repository.Repository{Owner: "owner", Name: "current"})
			v, valueErr := o.Values()
			if (err == nil) != (body == capBaseline) || (valueErr == nil) != (body == capBaseline) {
				t.Fatal(v, err, valueErr)
			}
		})
	}
}
func TestPullRequestLimitsUpdate(t *testing.T) {
	yes, no := true, false
	maximum := 9
	for _, tc := range []struct {
		name      string
		desired   settings.PullRequestLimitsDeclaration
		reads     []string
		status    int
		wantPatch map[string]any
		fail, pre bool
		calls     int
	}{
		{name: "enable preserves latent", desired: settings.PullRequestLimitsDeclaration{Enabled: &yes}, reads: []string{capBaseline, strings.Replace(capBaseline, `false`, `true`, 1)}, wantPatch: map[string]any{"enabled": true}, calls: 3},
		{name: "maximum only required enabled", desired: settings.PullRequestLimitsDeclaration{Maximum: &maximum}, reads: []string{capBaseline, strings.Replace(capBaseline, `:7`, `:9`, 1)}, wantPatch: map[string]any{"enabled": false, "max_open_pull_requests": float64(9)}, calls: 3},
		{name: "explicit false", desired: settings.PullRequestLimitsDeclaration{IncludeDrafts: &no}, reads: []string{capBaseline, strings.Replace(capBaseline, `drafts":true`, `drafts":false`, 1)}, wantPatch: map[string]any{"enabled": false, "include_drafts": false}, calls: 3},
		{name: "matching", desired: settings.PullRequestLimitsDeclaration{Enabled: &no}, reads: []string{capBaseline}, calls: 1},
		{name: "omitted", calls: 0},
		{name: "stale then match", desired: settings.PullRequestLimitsDeclaration{Enabled: &yes}, reads: []string{capBaseline, capBaseline, strings.Replace(capBaseline, `false`, `true`, 1)}, wantPatch: map[string]any{"enabled": true}, calls: 4},
		{name: "exhausted", desired: settings.PullRequestLimitsDeclaration{Enabled: &yes}, reads: []string{capBaseline, capBaseline, capBaseline, capBaseline}, wantPatch: map[string]any{"enabled": true}, calls: 5, fail: true},
		{name: "companion drift", desired: settings.PullRequestLimitsDeclaration{Enabled: &yes}, reads: []string{capBaseline, `{"enabled":true,"max_open_pull_requests":8,"include_drafts":true}`}, wantPatch: map[string]any{"enabled": true}, calls: 3, fail: true},
		{name: "malformed readback", desired: settings.PullRequestLimitsDeclaration{Enabled: &yes}, reads: []string{capBaseline, `{}`}, wantPatch: map[string]any{"enabled": true}, calls: 3, fail: true},
		{name: "denied preflight", desired: settings.PullRequestLimitsDeclaration{Enabled: &yes}, reads: []string{`{}`}, status: 403, calls: 1, fail: true, pre: true},
		{name: "denied readback", desired: settings.PullRequestLimitsDeclaration{Enabled: &yes}, reads: []string{capBaseline, `{}`}, status: 403, wantPatch: map[string]any{"enabled": true}, calls: 3, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, reads, patches, pauses := 0, 0, 0, 0
			c := &PullRequestLimitsClient{pause: func(time.Duration) { pauses++ }, rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/repos/owner/current/interaction-limits/pulls/creation-cap" {
					t.Fatal(req.URL)
				}
				if req.Method == "PATCH" {
					patches++
					var got map[string]any
					if json.NewDecoder(req.Body).Decode(&got) != nil || !reflect.DeepEqual(got, tc.wantPatch) {
						t.Fatal(got, tc.wantPatch)
					}
					return labelsResponse(req, 200, `{}`), nil
				}
				if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				body := tc.reads[reads]
				reads++
				status := 200
				if tc.status != 0 && ((tc.pre && reads == 1) || (!tc.pre && reads > 1)) {
					status = tc.status
				}
				return labelsResponse(req, status, body), nil
			})).rest}
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, tc.desired)
			var pre interface{ ReadFailedBeforeWrite() }
			if (err != nil) != tc.fail || errors.As(err, &pre) != tc.pre || calls != tc.calls || patches > 1 || wrote != (tc.wantPatch != nil && !tc.fail) {
				t.Fatalf("wrote=%v err=%v calls=%d patches=%d pauses=%d", wrote, err, calls, patches, pauses)
			}
		})
	}
}

func TestPullRequestLimitsWriteFailuresNeverRetry(t *testing.T) {
	enabled := true
	for _, status := range []int{0, 202, 302, 403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls, patches := 0, 0
			c := &PullRequestLimitsClient{rest: topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method == "GET" {
					return labelsResponse(req, 200, capBaseline), nil
				}
				patches++
				if status == 0 {
					return nil, errors.New("connection lost")
				}
				return labelsResponse(req, status, `{}`), nil
			})).rest}
			wrote, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, settings.PullRequestLimitsDeclaration{Enabled: &enabled})
			if err == nil || wrote || calls != 2 || patches != 1 {
				t.Fatal(wrote, err, calls, patches)
			}
		})
	}
}
