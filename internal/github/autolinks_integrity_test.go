package github

import (
	"net/http"
	"strings"
	"testing"
)

func TestAutolinkReadRejectsAmbiguousJSON(t *testing.T) {
	row := `[{"id":7,"key_prefix":"OLD-","url_template":"https://target.test/<num>?Case=A%2FB","is_alphanumeric":false}]`
	repo := `{"id":1,"full_name":"owner/current","private":false,"archived":true,"owner":{"id":2,"login":"owner","type":"User"}}`
	cases := []struct {
		name, list, target string
		good               bool
	}{
		{"complete", row, repo, true},
		{"unknown target metadata", row, strings.TrimSuffix(repo, "}") + `,"extra":{"safe":true}}`, true},
		{"public visibility", row, strings.Replace(repo, `"private":false`, `"private":false,"visibility":"public"`, 1), true},
		{"conflicting visibility", row, strings.Replace(repo, `"private":false`, `"private":false,"visibility":"private"`, 1), false},
		{"null visibility", row, strings.Replace(repo, `"private":false`, `"private":false,"visibility":null`, 1), false},
		{"visibility alias", row, strings.Replace(repo, `"private":false`, `"private":false,"Visibility":"public"`, 1), false},
		{"duplicate row id", strings.Replace(row, `"id":7`, `"id":8,"id":7`, 1), repo, false},
		{"escaped duplicate prefix", strings.Replace(row, `"key_prefix":"OLD-"`, `"key_prefix":"BAD-","key_\u0070refix":"OLD-"`, 1), repo, false},
		{"duplicate repo", row, strings.Replace(repo, `"id":1`, `"id":9,"id":1`, 1), false},
		{"repo alias", row, strings.Replace(repo, `"id":1`, `"ID":9,"id":1`, 1), false},
		{"owner alias", row, strings.Replace(repo, `"id":2`, `"ID":9,"id":2`, 1), false},
		{"nested duplicate", row, strings.TrimSuffix(repo, "}") + `,"extra":{"a":1,"a":2}}`, false},
		{"surrogate", strings.Replace(row, "OLD-", `\ud800`, 1), repo, false},
		{"invalid utf8", strings.Replace(row, "OLD-", string([]byte{0xff}), 1), repo, false},
		{"trailing", row + ` {}`, repo, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newAutolinkFixture(t)
			f.hook = func(req *http.Request, res *http.Response) {
				if strings.HasSuffix(req.URL.Path, "/autolinks") {
					*res = *labelsResponse(req, 200, tc.list)
				} else {
					*res = *labelsResponse(req, 200, tc.target)
				}
			}
			_, err := f.client.ReadAutolinks(autolinkRepo())
			if (err == nil) != tc.good {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestAutolinkReadRejectsAnyLinkHeader(t *testing.T) {
	f := newAutolinkFixture(t)
	f.hook = func(req *http.Request, res *http.Response) {
		if strings.HasSuffix(req.URL.Path, "/autolinks") {
			res.Header.Add("Link", "")
			res.Header.Add("Link", `<https://api.github.com/x>; rel="next"`)
		}
	}
	if _, err := f.client.ReadAutolinks(autolinkRepo()); err == nil {
		t.Fatal("accepted linked response")
	}
}

func TestAutolinkReadRejectsOversizedResponse(t *testing.T) {
	f := newAutolinkFixture(t)
	f.hook = func(req *http.Request, res *http.Response) {
		if strings.HasSuffix(req.URL.Path, "/autolinks") {
			*res = *labelsResponse(req, 200, `[`+strings.Repeat(" ", environmentResponseLimit)+`]`)
		}
	}
	if _, err := f.client.ReadAutolinks(autolinkRepo()); err == nil {
		t.Fatal("oversized response accepted")
	}
}
