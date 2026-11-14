package github

import (
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentPatternInventoryRejectsAmbiguousJSON(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantOK     bool
	}{
		{"complete empty", `{"total_count":0,"branch_policies":[]}`, true},
		{"duplicate count", `{"total_count":1,"total_count":0,"branch_policies":[]}`, false},
		{"duplicate list", `{"total_count":0,"branch_policies":[{"id":9,"type":"branch","name":"hidden"}],"branch_policies":[]}`, false},
		{"count case alias", `{"Total_Count":1,"total_count":0,"branch_policies":[]}`, false},
		{"list case alias", `{"total_count":0,"Branch_Policies":[{"id":9,"type":"branch","name":"hidden"}],"branch_policies":[]}`, false},
		{"unknown metadata", `{"total_count":0,"branch_policies":[],"future_metadata":"preserved as unrelated"}`, true},
		{"duplicate id", `{"total_count":1,"branch_policies":[{"id":8,"id":7,"type":"branch","name":"main"}]}`, false},
		{"duplicate type", `{"total_count":1,"branch_policies":[{"id":7,"type":"tag","type":"branch","name":"main"}]}`, false},
		{"duplicate name", `{"total_count":1,"branch_policies":[{"id":7,"type":"branch","name":"other","name":"main"}]}`, false},
		{"trailing", `{"total_count":0,"branch_policies":[]} {}`, false},
		{"malformed", `{"total_count":0,"branch_policies":[`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.Path != "/repos/owner/current/environments/production/deployment-branch-policies" {
					t.Fatalf("unexpected request %s %s", req.Method, req.URL)
				}
				return actionsResponse(req, http.StatusOK, tc.body), nil
			}).rest}
			states, err := client.readPatterns(repository.Repository{Owner: "owner", Name: "current"}, "production")
			if (err == nil) != tc.wantOK || calls != 1 {
				t.Fatalf("states=%v err=%v calls=%d", states, err, calls)
			}
			if tc.wantOK && states == nil {
				t.Fatal("a complete empty inventory must stay distinct from an unavailable read")
			}
		})
	}
}

func TestEnvironmentPatternAmbiguousRemovalEvidenceStopsBeforeDelete(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	current := []settings.EnvironmentPatternState{{ID: 7, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "old"}}}
	desired := []settings.EnvironmentPattern{}
	var methods []string
	client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		methods = append(methods, req.Method)
		return actionsResponse(req, http.StatusOK, `{"id":8,"id":7,"type":"branch","name":"old"}`), nil
	}).rest}
	attempted, err := client.applyPatterns(repo, "production", current, &desired, nil)
	if err == nil || attempted || strings.Join(methods, " ") != "GET" {
		t.Fatalf("attempted=%v err=%v methods=%v", attempted, err, methods)
	}
}

func TestEnvironmentPatternAmbiguousCreateAcknowledgmentStopsFurtherWrites(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	current := []settings.EnvironmentPatternState{{ID: 7, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "old"}}}
	desired := []settings.EnvironmentPattern{{Type: "branch", Name: "new"}}
	var methods []string
	client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		methods = append(methods, req.Method)
		return actionsResponse(req, http.StatusOK, `{"id":9,"type":"tag","type":"branch","name":"new"}`), nil
	}).rest}
	attempted, err := client.applyPatterns(repo, "production", current, &desired, nil)
	if err == nil || !attempted || !strings.Contains(err.Error(), "write may have applied") || strings.Join(methods, " ") != "POST" {
		t.Fatalf("attempted=%v err=%v methods=%v", attempted, err, methods)
	}
}

func TestEnvironmentPatternUnexpectedSuccessStatusFailsClosed(t *testing.T) {
	for _, operation := range []string{"inventory", "removal", "create"} {
		t.Run(operation, func(t *testing.T) {
			repo := repository.Repository{Owner: "owner", Name: "current"}
			current := []settings.EnvironmentPatternState{{ID: 7, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "old"}}}
			desired := []settings.EnvironmentPattern{}
			body := `{"id":7,"type":"branch","name":"old"}`
			if operation == "inventory" {
				body = `{"total_count":0,"branch_policies":[]}`
			}
			if operation == "create" {
				desired = []settings.EnvironmentPattern{{Type: "branch", Name: "new"}}
				body = `{"id":9,"type":"branch","name":"new"}`
			}
			var methods []string
			client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				methods = append(methods, req.Method)
				return actionsResponse(req, http.StatusPartialContent, body), nil
			}).rest}
			if operation == "inventory" {
				if _, err := client.readPatterns(repo, "production"); err == nil || strings.Join(methods, " ") != "GET" {
					t.Fatalf("err=%v methods=%v", err, methods)
				}
				return
			}
			attempted, err := client.applyPatterns(repo, "production", current, &desired, nil)
			if operation == "removal" {
				if err == nil || attempted || strings.Join(methods, " ") != "GET" {
					t.Fatalf("attempted=%v err=%v methods=%v", attempted, err, methods)
				}
			} else if err == nil || !attempted || !strings.Contains(err.Error(), "write may have applied") || strings.Join(methods, " ") != "POST" {
				t.Fatalf("attempted=%v err=%v methods=%v", attempted, err, methods)
			}
		})
	}
}
