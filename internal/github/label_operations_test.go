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
			if id != tc.wantID || (err != nil) != tc.wantError || calls != 1 {
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
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatal(calls)
			}
		})
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
