package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSingleLabelRequests(t *testing.T) {
	op := labelOperation(t)
	color, empty := "123456", ""
	before := settings.LabelState{Name: "bug", Color: "abcdef", Description: "old"}
	for _, test := range []struct {
		name         string
		call         func(*Client, repository.Repository, settings.LabelChange) error
		change       settings.LabelChange
		method, path string
		body         map[string]any
	}{
		{"name-only create", (*Client).CreateLabel, op(settings.NewLabelCreate(settings.LabelDeclaration{Name: "new"})), "POST", "/repos/owner/current/labels", map[string]any{"name": "new"}},
		{"explicit create", (*Client).CreateLabel, op(settings.NewLabelCreate(settings.LabelDeclaration{Name: "new", Color: &color, Description: &empty})), "POST", "/repos/owner/current/labels", map[string]any{"name": "new", "color": "123456", "description": ""}},
		{"color-only update", (*Client).UpdateLabel, op(settings.NewLabelUpdate(before, settings.LabelDeclaration{Name: "bug", Color: &color})), "PATCH", "/repos/owner/current/labels/bug", map[string]any{"color": "123456"}},
		{"empty description update", (*Client).UpdateLabel, op(settings.NewLabelUpdate(before, settings.LabelDeclaration{Name: "bug", Description: &empty})), "PATCH", "/repos/owner/current/labels/bug", map[string]any{"description": ""}},
		{"escaped delete", (*Client).DeleteLabel, op(settings.NewLabelDelete(settings.LabelState{Name: "area/docs ✨", Color: "abcdef"})), "DELETE", "/repos/owner/current/labels/area%2Fdocs%20%E2%9C%A8", nil},
		{"dot delete", (*Client).DeleteLabel, op(settings.NewLabelDelete(settings.LabelState{Name: ".", Color: "abcdef"})), "DELETE", "/repos/owner/current/labels/%2E", nil},
		{"double-dot delete", (*Client).DeleteLabel, op(settings.NewLabelDelete(settings.LabelState{Name: "..", Color: "abcdef"})), "DELETE", "/repos/owner/current/labels/%2E%2E", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := writeTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != test.method || req.URL.EscapedPath() != test.path {
					t.Fatalf("request=%s %s", req.Method, req.URL)
				}
				if test.body == nil {
					if req.Body != nil {
						t.Fatal("unexpected DELETE body")
					}
				} else {
					var body map[string]any
					if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(body, test.body) {
						t.Fatalf("body=%v want=%v", body, test.body)
					}
				}
				return labelsResponse(req, 204, ``), nil
			}))
			if err := test.call(client, repository.Repository{Owner: "owner", Name: "current"}, test.change); err != nil || calls != 1 {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
		})
	}
}

func TestSingleLabelMethodsRejectOtherOperationsBeforeRequests(t *testing.T) {
	op := labelOperation(t)
	create := op(settings.NewLabelCreate(settings.LabelDeclaration{Name: "new"}))
	remove := op(settings.NewLabelDelete(settings.LabelState{Name: "old", Color: "abcdef"}))
	client := writeTestClient(t, transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("request for wrong operation"); return nil, nil }))
	for _, test := range []struct {
		call  func(*Client, repository.Repository, settings.LabelChange) error
		wrong settings.LabelChange
	}{{(*Client).CreateLabel, remove}, {(*Client).UpdateLabel, create}, {(*Client).DeleteLabel, create}} {
		for _, change := range []settings.LabelChange{{}, test.wrong} {
			if err := test.call(client, repository.Repository{Owner: "owner", Name: "current"}, change); err == nil {
				t.Fatal("accepted wrong operation")
			}
		}
	}
}
