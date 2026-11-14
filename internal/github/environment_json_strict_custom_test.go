package github

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestEnvironmentJSONRejectsAmbiguityAndPreservesUnicode(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"duplicate top", `{"x":1,"x":2}`, false},
		{"duplicate nested", `{"x":[{"y":1,"y":2}]}`, false},
		{"trailing", `{"x":1} {}`, false},
		{"invalid utf8", "{\"x\":\"\xff\"}", false},
		{"unpaired high", `{"x":"\ud800"}`, false},
		{"unpaired low", `{"x":"\udc00"}`, false},
		{"invalid pair", `{"x":"\ud800\u0041"}`, false},
		{"pair", `{"x":"\ud83d\ude00"}`, true},
		{"literal backslash", `{"x":"\\ud800"}`, true},
		{"replacement", `{"x":"�"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var raw map[string]json.RawMessage
			err := environmentJSON([]byte(tc.body), &raw)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
}

func TestEnvironmentCustomInventoryRejectsAmbiguousEvidence(t *testing.T) {
	for _, body := range []string{
		`{"total_count":1,"total_count":0,"custom_deployment_protection_rules":[]}`,
		`{"total_count":0,"custom_deployment_protection_rules":[{}],"custom_deployment_protection_rules":[]}`,
		`{"total_count":1,"custom_deployment_protection_rules":[{"id":7,"node_id":"n","enabled":true,"app":{"id":8,"id":9,"node_id":"a","slug":"s","integration_url":"https://example.invalid"}}]}`,
	} {
		c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet {
				t.Fatal("unexpected mutation")
			}
			return actionsResponse(r, 200, body), nil
		}).rest}
		if _, err := c.readCustomRules(repository.Repository{Owner: "owner", Name: "current"}, "prod"); err == nil {
			t.Fatal("ambiguous custom protection inventory accepted")
		}
	}
}

func TestEnvironmentCustomRuleRecoveryRequiresUnambiguousID(t *testing.T) {
	for _, tc := range []struct {
		body string
		want int64
	}{
		{`{"id":102,"enabled":"invalid"}`, 102},
		{`{"id":102,"id":103,"enabled":true}`, 0},
		{`{"id":102,"ID":103,"enabled":true}`, 0},
		{`{"id":102,"enabled":true} {}`, 0},
		{`{"id":102,"unknown":"` + strings.Repeat("x", 100) + `"}`, 102},
	} {
		state, err := decodeEnvironmentCustomRule([]byte(tc.body))
		if err == nil || state.ID != tc.want {
			t.Fatalf("recovery ID=%d want=%d err=%v", state.ID, tc.want, err)
		}
	}
}
