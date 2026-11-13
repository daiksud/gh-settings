package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExportFixtureOnlyServesKnownObservations(t *testing.T) {
	for _, test := range []struct {
		name, method, path, body string
		allowed                  bool
	}{
		{"variable read", "GET", "/repos/example-owner/settings-fixture/actions/variables?per_page=30&page=1", "", true},
		{"unavailable read", "GET", "/repos/example-owner/settings-fixture/pages", "", true},
		{"read query", "POST", "/graphql", `{"query":"query($owner: String!, $name: String!) { repository(owner: $owner, name: $name, followRenames: false) { issueCreationPolicy } }","variables":{"owner":"example-owner","name":"settings-fixture"}}`, true},
		{"REST mutation", "PATCH", "/repos/example-owner/settings-fixture", `{}`, false},
		{"GraphQL mutation", "POST", "/graphql", `{"query":"mutation { ignored }","variables":{"owner":"example-owner","name":"settings-fixture"}}`, false},
		{"unknown query", "POST", "/graphql", `{"query":"query { viewer { login } }","variables":{"owner":"example-owner","name":"settings-fixture"}}`, false},
		{"other target", "GET", "/repos/example-owner/other", "", false},
		{"unlisted read", "GET", "/repos/example-owner/settings-fixture/contents/private", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "https://api.github.com"+test.path, strings.NewReader(test.body))
			var log bytes.Buffer
			response := httptest.NewRecorder()
			if got := serveExport(response, request, &log); got != test.allowed {
				t.Fatalf("handled=%t want=%t", got, test.allowed)
			}
			if !test.allowed && (log.Len() != 0 || response.Body.Len() != 0) {
				t.Fatal("rejected request was served or recorded as an observation")
			}
			if test.name == "variable read" && (response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "CANARY-local-export-value") || strings.Contains(log.String(), "CANARY-local-export-value")) {
				t.Fatal("variable fixture lost response-only canary")
			}
		})
	}
}
