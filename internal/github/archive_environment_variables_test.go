package github

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestArchivedEnvironmentVariablesParentBindsSelectedReads(t *testing.T) {
	const canary = "sensitive-fixture-canary"
	for _, scenario := range []string{"matching", "parent drift", "parent denied", "parent malformed", "variable denied", "variable malformed", "absent"} {
		t.Run(scenario, func(t *testing.T) {
			var paths []string
			parentReads := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Fatalf("unexpected write %s", req.Method)
				}
				paths = append(paths, req.URL.Path)
				if req.URL.Path == "/repos/owner/current/environments/prod" {
					parentReads++
					if parentReads == 2 && scenario == "parent denied" {
						return nil, errors.New(canary)
					}
					if parentReads == 2 && scenario == "parent malformed" {
						return labelsResponse(req, 200, `{"id":42,"id":42,"name":"prod"}`), nil
					}
					if parentReads == 2 && scenario == "parent drift" {
						return labelsResponse(req, 200, `{"id":43,"name":"prod"}`), nil
					}
					return labelsResponse(req, 200, `{"id":42,"name":"prod"}`), nil
				}
				if req.URL.Path == "/repos/owner/current/environments/prod/variables/A" {
					return labelsResponse(req, 200, `{"name":"A","value":"`+canary+`"}`), nil
				}
				if req.URL.Path == "/repos/owner/current/environments/prod/variables/B" {
					switch scenario {
					case "variable denied":
						return nil, errors.New(canary)
					case "variable malformed":
						return labelsResponse(req, 200, `{"name":"B","value":"`+canary+`","value":"x"}`), nil
					case "absent":
						return labelsResponse(req, 404, `{}`), nil
					}
					return labelsResponse(req, 200, `{"name":"B","value":"`+canary+`"}`), nil
				}
				t.Fatalf("unexpected path %s", req.URL.Path)
				return nil, nil
			}))
			id, values, err := client.ReadArchivedEnvironmentVariables(repository.Repository{Owner: "owner", Name: "current"}, "prod", []string{"A", "B"})
			failed := scenario != "matching" && scenario != "absent"
			if (err != nil) != failed || err != nil && strings.Contains(err.Error(), canary) {
				t.Fatalf("unexpected error: %v", err)
			}
			if !failed && (id != 42 || len(values) != 2 || parentReads != 2 || len(paths) != 4) {
				t.Fatalf("id=%d count=%d parents=%d paths=%v", id, len(values), parentReads, paths)
			}
			if scenario == "absent" {
				v, e := values[1].Value()
				if e != nil || v != nil {
					t.Fatalf("absent observation not retained")
				}
			}
		})
	}
}
