package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUILabelPresenceAndValidation(t *testing.T) {
	if got, err := LabelsDeclaration(nil); err != nil || got != nil {
		t.Fatal(got, err)
	}
	empty := []uimodel.Label{}
	if got, err := LabelsDeclaration(&empty); err != nil || got == nil || got.Len() != 0 {
		t.Fatal(got, err)
	}
	for _, color := range []string{"abcdef", "#abc", "#zzzzzz", ""} {
		labels := []uimodel.Label{{Name: "Example", Color: pointer(color)}}
		if _, err := LabelsDeclaration(&labels); err == nil {
			t.Fatalf("invalid UI color accepted: %q", color)
		}
	}
	if _, err := LabelsObservation(settings.LabelObservation{}); err == nil {
		t.Fatal("unacquired labels accepted")
	}
}

func TestUILabelReadAndSelectedUpdate(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	calls := 0
	client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			if req.Method != "GET" || req.URL.Path != "/repos/owner/current/labels" {
				t.Fatal(req.Method, req.URL)
			}
			return labelsResponse(req, 200, `[{"id":7,"name":"Old","color":"AABBCC","description":null},{"id":8,"name":"Unlisted","color":"123456","description":"retain"}]`), nil
		case 2:
			if req.Method != "GET" || req.URL.Path != "/repos/owner/current/labels/Old" {
				t.Fatal(req.Method, req.URL)
			}
			return labelsResponse(req, 200, `{"id":7}`), nil
		case 3:
			want := map[string]any{"new_name": "New", "color": "00ff11", "description": ""}
			var got map[string]any
			if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if req.Method != "PATCH" || req.URL.Path != "/repos/owner/current/labels/Old" || !reflect.DeepEqual(got, want) {
				t.Fatalf("payload=%v want %v", got, want)
			}
			return labelsResponse(req, 200, `{"id":7,"name":"New","color":"00ff11","description":""}`), nil
		default:
			t.Fatal("unexpected request")
			return nil, nil
		}
	})
	observed, err := client.ReadLabels(repo)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LabelsObservation(observed)
	want := []uimodel.Label{{ID: pointer(int64(7)), Name: "Old", Color: pointer("#aabbcc"), Description: pointer("")}, {ID: pointer(int64(8)), Name: "Unlisted", Color: pointer("#123456"), Description: pointer("retain")}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("labels=%+v err=%v want %+v", got, err, want)
	}
	labels := []uimodel.Label{{ID: pointer(int64(7)), Name: "New", Color: pointer("#00FF11")}}
	desired, err := LabelsDeclaration(&labels)
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := settings.CompareLabels(observed, desired)
	if err != nil || comparison.Labels.Len() != 1 {
		t.Fatal(comparison, err)
	}
	for _, change := range comparison.Labels.Values() {
		if change.Operation() != "update" {
			t.Fatal(change.Operation())
		}
		if err := client.UpdateLabel(repo, change); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Fatal(calls)
	}
}

func TestUILabelCreateAndInvalidIDs(t *testing.T) {
	for _, ids := range [][]int64{{0}, {-1}, {7, 7}} {
		labels := []uimodel.Label{}
		for i, id := range ids {
			labels = append(labels, uimodel.Label{ID: pointer(id), Name: []string{"One", "Two"}[i], Color: pointer("#abcdef")})
		}
		if _, err := LabelsDeclaration(&labels); err == nil {
			t.Fatal("invalid label IDs accepted", ids)
		}
	}
	labels := []uimodel.Label{{Name: "New", Color: pointer("#ABCDEF")}}
	desired, err := LabelsDeclaration(&labels)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := settings.NewLabelObservation(nil)
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := settings.CompareLabels(empty, desired)
	if err != nil || comparison.Labels.Len() != 1 {
		t.Fatal(comparison, err)
	}
	client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
		want := map[string]any{"name": "New", "color": "abcdef", "description": ""}
		var got map[string]any
		if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if req.Method != "POST" || req.URL.Path != "/repos/owner/current/labels" || !reflect.DeepEqual(got, want) {
			t.Fatalf("payload=%v want %v", got, want)
		}
		return labelsResponse(req, 200, `{"id":9,"name":"New","color":"abcdef","description":""}`), nil
	})
	for _, change := range comparison.Labels.Values() {
		id, err := client.CreateLabel(repository.Repository{Owner: "owner", Name: "current"}, change)
		if err != nil || id != 9 {
			t.Fatal(id, err)
		}
	}
}
