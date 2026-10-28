package github

import (
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIAboutDeclaration(t *testing.T) {
	for _, test := range []struct {
		name string
		ui   uimodel.About
		want settings.Declaration
	}{
		{name: "omitted"},
		{name: "empty", ui: uimodel.About{Description: pointer(""), Website: pointer(""), Topics: pointer([]string{})}, want: settings.Declaration{Repository: settings.RepositoryDeclaration{Description: pointer(""), Homepage: pointer("")}, Topics: settings.TopicsDeclaration{Names: pointer([]string{})}}},
		{name: "values", ui: uimodel.About{Description: pointer("Example"), Website: pointer("https://example.com"), Topics: pointer([]string{"tools", "github"})}, want: settings.Declaration{Repository: settings.RepositoryDeclaration{Description: pointer("Example"), Homepage: pointer("https://example.com")}, Topics: settings.TopicsDeclaration{Names: pointer([]string{"github", "tools"})}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := AboutDeclaration(test.ui)
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("declaration=%+v error=%v, want %+v", got, err, test.want)
			}
		})
	}
}

func TestUIAboutObservation(t *testing.T) {
	client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" || (req.URL.Path != "/repos/owner/current" && req.URL.Path != "/repos/owner/current/topics") {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
		}
		body := `{"description":null,"homepage":"https://example.com","default_branch":"main","has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`
		if strings.HasSuffix(req.URL.Path, "/topics") {
			body = `{"names":["tools","github"]}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	state, err := client.ReadRepository(repo)
	if err != nil {
		t.Fatal(err)
	}
	topics, err := client.ReadTopics(repo)
	if err != nil {
		t.Fatal(err)
	}
	got := AboutObservation(state, topics)
	want := uimodel.About{Description: pointer(""), Website: pointer("https://example.com"), Topics: pointer([]string{"github", "tools"})}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("About=%+v, want %+v", got, want)
	}
}

func TestUIAboutOwnsValues(t *testing.T) {
	ui := uimodel.About{Description: pointer("Example"), Website: pointer("https://example.com"), Topics: pointer([]string{"tools"})}
	got, err := AboutDeclaration(ui)
	if err != nil {
		t.Fatal(err)
	}
	*ui.Description = "changed"
	*ui.Website = "changed"
	(*ui.Topics)[0] = "changed"
	if *got.Repository.Description != "Example" || *got.Repository.Homepage != "https://example.com" || (*got.Topics.Names)[0] != "tools" {
		t.Fatal("conversion aliases caller values")
	}
}

func TestUIAboutRejectsInvalidTopics(t *testing.T) {
	for _, topics := range [][]string{{"UPPER"}, {"same", "same"}} {
		if _, err := AboutDeclaration(uimodel.About{Topics: &topics}); err == nil {
			t.Fatal("invalid topics accepted")
		}
	}
}

func TestUIAboutEmptyPayloads(t *testing.T) {
	calls := 0
	client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		var got map[string]any
		if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{"description": "", "homepage": ""}
		if calls == 1 {
			if req.Method != "PATCH" || req.URL.Path != "/repos/owner/current" {
				t.Fatal(req.Method, req.URL)
			}
		} else {
			want = map[string]any{"names": []any{}}
			if calls != 2 || req.Method != "PUT" || req.URL.Path != "/repos/owner/current/topics" {
				t.Fatal(req.Method, req.URL)
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("payload=%v want %v", got, want)
		}
		return labelsResponse(req, 200, `{}`), nil
	})
	desired, err := AboutDeclaration(uimodel.About{Description: pointer(""), Website: pointer(""), Topics: pointer([]string{})})
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.Repository{Owner: "owner", Name: "current"}
	if err := client.ApplyRepository(repo, desired.Repository); err != nil {
		t.Fatal(err)
	}
	if err := client.ApplyTopics(repo, desired.Topics); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}

func pointer[T any](v T) *T { return &v }
