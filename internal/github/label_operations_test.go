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
)

func TestLabelCreateReturnsVerifiedID(t *testing.T) {
	color, description := "123456", ""
	change, err := settings.NewLabelCreate(settings.LabelDeclaration{Name: "new", Color: &color, Description: &description})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, body string
		wantID     int64
		wantError  bool
	}{
		{"complete", `{"id":17,"name":"new","color":"123456","description":null}`, 17, false},
		{"missing ID", `{"name":"new","color":"123456","description":""}`, 0, true},
		{"mismatching response", `{"id":17,"name":"wrong","color":"123456","description":""}`, 17, true},
		{"missing value", `{"id":17,"name":"new","color":"123456"}`, 17, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := writeTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method == "GET" && req.URL.Path == "/repos/owner/current/labels/new" && calls == 2 && !tc.wantError {
					return labelsResponse(req, 200, `{"id":17,"name":"new","color":"123456","description":""}`), nil
				}
				if req.Method != "POST" || req.URL.Path != "/repos/owner/current/labels" {
					t.Fatal(req.Method, req.URL)
				}
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(payload, map[string]any{"name": "new", "color": "123456", "description": ""}) {
					t.Fatal(payload)
				}
				return labelsResponse(req, 201, tc.body), nil
			}))
			id, err := client.CreateLabel(repository.Repository{Owner: "owner", Name: "current"}, change)
			wantCalls := 1
			if !tc.wantError {
				wantCalls++
			}
			if id != tc.wantID || (err != nil) != tc.wantError || calls != wantCalls {
				t.Fatal(id, err, calls)
			}
		})
	}
}

func TestLabelUpdatePinsIDAndRenamesWithoutRecreate(t *testing.T) {
	color, description, id := "123456", "", int64(7)
	change, err := settings.NewLabelUpdate(settings.LabelState{ID: 7, Name: "area/docs ✨", Color: "abcdef", Description: "old"}, settings.LabelDeclaration{ID: &id, Name: "renamed", Color: &color, Description: &description})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                   string
		observedID, responseID int64
		status                 int
		wantError              bool
	}{
		{"match", 7, 7, 200, false}, {"replaced before write", 8, 7, 200, true}, {"replaced during write", 7, 8, 200, true}, {"denied", 7, 7, 403, true}, {"redirect", 7, 7, 307, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			client := writeTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls = append(calls, req.Method)
				if req.Method == "GET" && len(calls) == 3 && req.URL.Path == "/repos/owner/current/labels/renamed" {
					return labelsResponse(req, 200, `{"id":7,"name":"renamed","color":"123456","description":""}`), nil
				}
				if req.URL.EscapedPath() != "/repos/owner/current/labels/area%2Fdocs%20%E2%9C%A8" {
					t.Fatal(req.URL)
				}
				if req.Method == "GET" {
					return labelsResponse(req, 200, fmt.Sprintf(`{"id":%d}`, tc.observedID)), nil
				}
				if req.Method != "PATCH" {
					t.Fatal("unexpected mutation", req.Method)
				}
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(payload, map[string]any{"new_name": "renamed", "color": "123456", "description": ""}) {
					t.Fatal(payload)
				}
				return labelsResponse(req, tc.status, fmt.Sprintf(`{"id":%d,"name":"renamed","color":"123456","description":""}`, tc.responseID)), nil
			}))
			err := client.UpdateLabel(repository.Repository{Owner: "owner", Name: "current"}, change)
			if (err != nil) != tc.wantError {
				t.Fatal(err)
			}
			var beforeWrite interface{ ReadFailedBeforeWrite() }
			if errors.As(err, &beforeWrite) != (tc.observedID != 7) {
				t.Fatalf("incorrect pre-write failure classification: %v", err)
			}
			want := []string{"GET", "PATCH"}
			if tc.observedID != 7 {
				want = want[:1]
			} else if !tc.wantError {
				want = append(want, "GET")
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatal(calls)
			}
		})
	}
}

func TestLabelMutationsRequireIndependentCompleteReadback(t *testing.T) {
	const complete = `{"id":17,"name":"new /✨","color":"abcdef","description":""}`
	cause := errors.New("readback connection lost")
	for _, operation := range []string{"create", "update"} {
		for _, tc := range []struct {
			name, body string
			status     int
			wantError  bool
		}{
			{"complete", complete, 200, false},
			{"normalized", `{"id":17,"name":"NEW /✨","color":"ABCDEF","description":null}`, 200, false},
			{"wrong ID", `{"id":18,"name":"new /✨","color":"abcdef","description":""}`, 200, true},
			{"missing ID", `{"name":"new /✨","color":"abcdef","description":""}`, 200, true},
			{"wrong ID type", `{"id":"17","name":"new /✨","color":"abcdef","description":""}`, 200, true},
			{"stale name", `{"id":17,"name":"old","color":"abcdef","description":""}`, 200, true},
			{"missing name", `{"id":17,"color":"abcdef","description":""}`, 200, true},
			{"null name", `{"id":17,"name":null,"color":"abcdef","description":""}`, 200, true},
			{"stale color", `{"id":17,"name":"new /✨","color":"000000","description":""}`, 200, true},
			{"null color", `{"id":17,"name":"new /✨","color":null,"description":""}`, 200, true},
			{"missing color", `{"id":17,"name":"new /✨","description":""}`, 200, true},
			{"stale description", `{"id":17,"name":"new /✨","color":"abcdef","description":"old"}`, 200, true},
			{"missing description", `{"id":17,"name":"new /✨","color":"abcdef"}`, 200, true},
			{"wrong description type", `{"id":17,"name":"new /✨","color":"abcdef","description":false}`, 200, true},
			{"null object", `null`, 200, true},
			{"wrong body type", `[]`, 200, true},
			{"malformed body", `{`, 200, true},
			{"denied", `{"message":"denied"}`, 403, true},
			{"missing", `{"message":"not found"}`, 404, true},
			{"redirect", complete, 307, true},
			{"transport", "", 0, true},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				color, description, id := "abcdef", "", int64(17)
				after := settings.LabelDeclaration{Name: "new /✨", Color: &color, Description: &description}
				change, err := settings.NewLabelCreate(after)
				wantCalls := []string{"POST /repos/owner/current/labels", "GET /repos/owner/current/labels/new%20%2F%E2%9C%A8"}
				if operation == "update" {
					after.ID = &id
					change, err = settings.NewLabelUpdate(settings.LabelState{ID: id, Name: "old", Color: color}, after)
					wantCalls = []string{"GET /repos/owner/current/labels/old", "PATCH /repos/owner/current/labels/old", "GET /repos/owner/current/labels/new%20%2F%E2%9C%A8"}
				}
				if err != nil {
					t.Fatal(err)
				}
				var calls []string
				client := writeTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					calls = append(calls, req.Method+" "+req.URL.EscapedPath())
					if req.Method == "GET" && req.URL.Path == "/repos/owner/current/labels/old" {
						return labelsResponse(req, 200, `{"id":17}`), nil
					}
					if req.Method != "GET" {
						return labelsResponse(req, 200, complete), nil
					}
					if tc.status == 0 {
						return nil, cause
					}
					response := labelsResponse(req, tc.status, tc.body)
					if tc.status == http.StatusTemporaryRedirect {
						response.Header.Set("Location", "https://api.github.com/repos/other/wrong/labels/new")
					}
					return response, nil
				}))
				repo := repository.Repository{Owner: "owner", Name: "current"}
				if operation == "create" {
					var actualID int64
					actualID, err = client.CreateLabel(repo, change)
					if actualID != id {
						t.Fatalf("lost recovery ID: %d, error=%v", actualID, err)
					}
				} else {
					err = client.UpdateLabel(repo, change)
				}
				if (err != nil) != tc.wantError || !reflect.DeepEqual(calls, wantCalls) {
					t.Fatalf("error=%v calls=%v want=%v", err, calls, wantCalls)
				}
				var beforeWrite interface{ ReadFailedBeforeWrite() }
				if errors.As(err, &beforeWrite) || (tc.wantError && !strings.Contains(err.Error(), "read back label (write may have applied)")) || (tc.status == 0 && !errors.Is(err, cause)) {
					t.Fatalf("incorrect postwrite failure: %v", err)
				}
			})
		}
	}
}

func TestSingleLabelMethodsRejectOtherOperationsBeforeRequests(t *testing.T) {
	color := "abcdef"
	create, err := settings.NewLabelCreate(settings.LabelDeclaration{Name: "new", Color: &color})
	if err != nil {
		t.Fatal(err)
	}
	client := writeTestClient(t, transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected request"); return nil, nil }))
	if _, err := client.CreateLabel(repository.Repository{}, settings.LabelChange{}); err == nil {
		t.Fatal("invalid create accepted")
	}
	if err := client.UpdateLabel(repository.Repository{}, create); err == nil || !strings.Contains(err.Error(), "update") {
		t.Fatal(err)
	}
}
