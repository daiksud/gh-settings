package github

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestVariableInventoryRejectsAmbiguousPages(t *testing.T) {
	const private = "PRIVATE_VARIABLE_CANARY"
	for _, family := range []string{"actions", "agents", "environment"} {
		for _, tc := range []struct {
			name, body string
		}{
			{"duplicate total", `{"total_count":1,"total_count":0,"variables":[]}`},
			{"total alias", `{"total_count":0,"TOTAL_COUNT":1,"variables":[]}`},
			{"duplicate list", `{"total_count":0,"variables":[{}],"variables":[]}`},
			{"list alias", `{"total_count":0,"variables":[],"Variables":[{}]}`},
			{"duplicate name", `{"total_count":1,"variables":[{"name":"WRONG","name":"ITEM","value":"` + private + `"}]}`},
			{"name alias", `{"total_count":1,"variables":[{"name":"ITEM","Name":"WRONG","value":"` + private + `"}]}`},
			{"duplicate value", `{"total_count":1,"variables":[{"name":"ITEM","value":"other","value":"` + private + `"}]}`},
			{"value alias", `{"total_count":1,"variables":[{"name":"ITEM","value":"` + private + `","Value":"other"}]}`},
			{"nested duplicate", `{"total_count":0,"variables":[],"metadata":{"x":1,"x":2}}`},
			{"null entry", `{"total_count":1,"variables":[null]}`},
			{"invalid utf8", "{\"total_count\":1,\"variables\":[{\"name\":\"ITEM\",\"value\":\"\xff\"}]}"},
			{"unpaired high", `{"total_count":1,"variables":[{"name":"ITEM","value":"\ud800"}]}`},
			{"unpaired low", `{"total_count":1,"variables":[{"name":"ITEM","value":"\udc00"}]}`},
			{"trailing", `{"total_count":0,"variables":[]} {}`},
			{"oversized", `{"total_count":0,"variables":[],"padding":"` + strings.Repeat("x", 16<<20) + `"}`},
		} {
			t.Run(family+"/"+tc.name, func(t *testing.T) {
				listReads, writes := 0, 0
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodGet {
						writes++
						return labelsResponse(req, 500, `{}`), nil
					}
					if req.URL.Path == "/repos/owner/current/environments/prod" {
						return labelsResponse(req, 200, `{"id":42,"name":"prod"}`), nil
					}
					listReads++
					return labelsResponse(req, 200, tc.body), nil
				}))
				repo := repository.Repository{Owner: "owner", Name: "current"}
				var observed settings.VariableInventory
				var err error
				switch family {
				case "actions":
					observed, err = client.ReadVariables(repo)
				case "agents":
					observed, err = client.ReadAgentVariables(repo)
				case "environment":
					observed, err = client.ReadEnvironmentVariables(repo, "prod")
				}
				_, valueErr := observed.Values()
				if err == nil || valueErr == nil || listReads != 1 || writes != 0 || strings.Contains(err.Error(), private) {
					t.Fatalf("err=%v observation error=%v reads=%d writes=%d", err, valueErr, listReads, writes)
				}
			})
		}
	}
}

func TestVariableInventoryValidUnicodeAndEmptyValues(t *testing.T) {
	client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		return labelsResponse(req, 200, `{"total_count":4,"variables":[{"name":"EMPTY","value":""},{"name":"PAIR","value":"\ud83d\ude00"},{"name":"BACKSLASH","value":"\\ud800"},{"name":"REPLACEMENT","value":"�"}]}`), nil
	}))
	observed, err := client.ReadVariables(repository.Repository{Owner: "owner", Name: "current"})
	if err != nil {
		t.Fatal(err)
	}
	values, err := observed.Values()
	if err != nil || len(values) != 4 {
		t.Fatalf("values=%d err=%v", len(values), err)
	}
	want := map[string]string{"EMPTY": "", "PAIR": "😀", "BACKSLASH": `\ud800`, "REPLACEMENT": "�"}
	for _, value := range values {
		if want[value.Name] != value.Value {
			t.Fatal("valid Unicode or empty value changed")
		}
	}
}

func TestVariableInventoryRejectsUnavailableAndPartialStatus(t *testing.T) {
	for _, status := range []int{http.StatusPartialContent, http.StatusFound, http.StatusForbidden, http.StatusNotFound} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				return labelsResponse(req, status, `{"total_count":0,"variables":[]}`), nil
			}))
			observed, err := client.ReadVariables(repository.Repository{Owner: "owner", Name: "current"})
			_, observedErr := observed.Values()
			if err == nil || observedErr == nil {
				t.Fatalf("status=%d err=%v observed=%v", status, err, observedErr)
			}
		})
	}
}

func TestVariableDeletionRejectsAmbiguousReadback(t *testing.T) {
	for _, family := range []string{"actions", "agents", "environment"} {
		t.Run(family, func(t *testing.T) {
			deletes, inventoryReads := 0, 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/repos/owner/current/environments/prod" {
					return labelsResponse(req, 200, `{"id":42,"name":"prod"}`), nil
				}
				if req.Method == http.MethodDelete {
					deletes++
					return labelsResponse(req, 204, ""), nil
				}
				if strings.HasSuffix(req.URL.Path, "/OLD") {
					return labelsResponse(req, 200, `{"name":"OLD","value":"before"}`), nil
				}
				inventoryReads++
				return labelsResponse(req, 200, `{"total_count":1,"total_count":0,"variables":[{"name":"OLD","value":"before"}],"variables":[]}`), nil
			}))
			repo := repository.Repository{Owner: "owner", Name: "current"}
			change, err := settings.NewVariableDelete(settings.RepositoryVariable{Name: "OLD", Value: "before"})
			if err != nil {
				t.Fatal(err)
			}
			switch family {
			case "actions":
				err = client.DeleteVariable(repo, change)
			case "agents":
				err = client.DeleteAgentVariable(repo, change)
			case "environment":
				err = client.DeleteEnvironmentVariable(repo, "prod", 42, change)
			}
			if err == nil || !strings.Contains(err.Error(), "write may have applied") || deletes != 1 || inventoryReads != 1 {
				t.Fatalf("err=%v deletes=%d inventory reads=%d", err, deletes, inventoryReads)
			}
		})
	}
}

func TestVariableInventorySecondPageCannotBecomePartial(t *testing.T) {
	reads := 0
	client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		reads++
		if reads == 1 {
			entries := make([]string, 30)
			for i := range entries {
				entries[i] = fmt.Sprintf(`{"name":"ITEM_%d","value":"present"}`, i)
			}
			return labelsResponse(req, 200, `{"total_count":31,"variables":[`+strings.Join(entries, ",")+`]}`), nil
		}
		return labelsResponse(req, 200, `{"total_count":31,"total_count":31,"variables":[{"name":"LAST","value":"present"}]}`), nil
	}))
	observed, err := client.ReadVariables(repository.Repository{Owner: "owner", Name: "current"})
	_, observedErr := observed.Values()
	if err == nil || observedErr == nil || reads != 2 {
		t.Fatalf("err=%v observed err=%v reads=%d", err, observedErr, reads)
	}
}

func TestVariableInventoryRejectsPaginationContradictions(t *testing.T) {
	entries := make([]string, 31)
	for i := range entries {
		entries[i] = fmt.Sprintf(`{"name":"ITEM_%d","value":"present"}`, i)
	}
	for _, family := range []string{"actions", "agents", "environment"} {
		for _, tc := range []struct {
			name, body string
			links      []string
		}{
			{"next link on completed empty inventory", `{"total_count":0,"variables":[]}`, []string{`<https://example.invalid/next>; rel="next"`}},
			{"next in second link header", `{"total_count":0,"variables":[]}`, []string{`<https://example.invalid/last>; rel="last"`, `<https://example.invalid/next>; rel="next"`}},
			{"malformed link", `{"total_count":0,"variables":[]}`, []string{`not-a-link`}},
			{"empty link", `{"total_count":0,"variables":[]}`, []string{""}},
			{"unfinished count with prev only", `{"total_count":31,"variables":[` + strings.Join(entries[:30], ",") + `]}`, []string{`<https://example.invalid/prev>; rel="prev"`}},
			{"overfull terminal page", `{"total_count":31,"variables":[` + strings.Join(entries, ",") + `]}`, nil},
		} {
			for _, stage := range []string{"preflight", "readback"} {
				t.Run(family+"/"+tc.name+"/"+stage, func(t *testing.T) {
					deletes, lists, individual, unexpected := 0, 0, 0, 0
					client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
						if req.URL.Host != "github.com" && req.URL.Host != "api.github.com" {
							unexpected++
						}
						if req.URL.Path == "/repos/owner/current/environments/prod" {
							return labelsResponse(req, 200, `{"id":42,"name":"prod"}`), nil
						}
						if req.Method == http.MethodDelete {
							deletes++
							return labelsResponse(req, 204, ""), nil
						}
						if strings.HasSuffix(req.URL.Path, "/OLD") {
							individual++
							return labelsResponse(req, 200, `{"name":"OLD","value":"before"}`), nil
						}
						lists++
						if tc.name == "unfinished count with prev only" && lists > 1 {
							return labelsResponse(req, 200, `{"total_count":31,"variables":[{"name":"LAST","value":"present"}]}`), nil
						}
						resp := labelsResponse(req, 200, tc.body)
						for _, link := range tc.links {
							resp.Header.Add("Link", link)
						}
						return resp, nil
					}))
					repo := repository.Repository{Owner: "owner", Name: "current"}
					var err error
					if stage == "preflight" {
						var inventory settings.VariableInventory
						switch family {
						case "actions":
							inventory, err = client.ReadVariables(repo)
						case "agents":
							inventory, err = client.ReadAgentVariables(repo)
						case "environment":
							inventory, err = client.ReadEnvironmentVariables(repo, "prod")
						}
						_, valueErr := inventory.Values()
						if err == nil || valueErr == nil || deletes != 0 || lists != 1 || individual != 0 || unexpected != 0 {
							t.Fatalf("err=%v observation=%v deletes=%d lists=%d individual=%d unexpected=%d", err, valueErr, deletes, lists, individual, unexpected)
						}
					} else {
						change, changeErr := settings.NewVariableDelete(settings.RepositoryVariable{Name: "OLD", Value: "before"})
						if changeErr != nil {
							t.Fatal(changeErr)
						}
						switch family {
						case "actions":
							err = client.DeleteVariable(repo, change)
						case "agents":
							err = client.DeleteAgentVariable(repo, change)
						case "environment":
							err = client.DeleteEnvironmentVariable(repo, "prod", 42, change)
						}
						if err == nil || !strings.Contains(err.Error(), "write may have applied") || deletes != 1 || lists != 1 || individual != 1 || unexpected != 0 {
							t.Fatalf("err=%v deletes=%d lists=%d individual=%d unexpected=%d", err, deletes, lists, individual, unexpected)
						}
					}
				})
			}
		}
	}
}

func TestVariableInventoryAcceptsConsistentNextPage(t *testing.T) {
	reads := 0
	client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		reads++
		if reads == 1 {
			entries := make([]string, 30)
			for i := range entries {
				entries[i] = fmt.Sprintf(`{"name":"ITEM_%d","value":"present"}`, i)
			}
			resp := labelsResponse(req, 200, `{"total_count":31,"variables":[`+strings.Join(entries, ",")+`]}`)
			resp.Header.Set("Link", `<https://example.invalid/untrusted?page=2>; rel="next", <https://example.invalid/untrusted?page=2>; rel="last"`)
			return resp, nil
		}
		if req.URL.Host != "api.github.com" || req.URL.Query().Get("page") != "2" {
			t.Fatalf("pagination did not construct expected local path")
		}
		resp := labelsResponse(req, 200, `{"total_count":31,"variables":[{"name":"LAST","value":"present"}]}`)
		resp.Header.Set("Link", `<https://example.invalid/untrusted?page=1>; rel="prev", <https://example.invalid/untrusted?page=1>; rel="first"`)
		return resp, nil
	}))
	inventory, err := client.ReadVariables(repository.Repository{Owner: "owner", Name: "current"})
	if err != nil {
		t.Fatal(err)
	}
	values, err := inventory.Values()
	if err != nil || reads != 2 || len(values) != 31 {
		t.Fatalf("err=%v reads=%d count=%d", err, reads, len(values))
	}
}

// The 16 MiB bound accommodates 30 values of 48 KiB even if each byte is
// represented by a six-byte JSON escape (under 9 MiB plus response metadata).
func TestVariableInventoryPageSizeBoundaries(t *testing.T) {
	prefix, suffix := `{"total_count":0,"variables":[],"padding":"`, `"}`
	for _, extra := range []int{0, 1} {
		size := variableInventoryPageLimit + extra
		body := prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				return labelsResponse(req, 200, body), nil
			}))
			inventory, err := client.ReadVariables(repository.Repository{Owner: "owner", Name: "current"})
			if extra == 0 {
				values, valueErr := inventory.Values()
				if err != nil || valueErr != nil || len(values) != 0 {
					t.Fatalf("at limit: err=%v values=%d valueErr=%v", err, len(values), valueErr)
				}
			} else if err == nil {
				t.Fatal("accepted oversized page")
			}
		})
	}
	entries := make([]string, 30)
	for i := range entries {
		entries[i] = fmt.Sprintf(`{"name":"ITEM_%d","value":"%s"}`, i, strings.Repeat(`\u0001`, 48<<10))
	}
	body := `{"total_count":30,"variables":[` + strings.Join(entries, ",") + `]}`
	if len(body) >= variableInventoryPageLimit {
		t.Fatal("representative page exceeds limit")
	}
	client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, body), nil }))
	inventory, err := client.ReadVariables(repository.Repository{Owner: "owner", Name: "current"})
	if err != nil {
		t.Fatalf("representative escaped page: %v", err)
	}
	values, err := inventory.Values()
	if err != nil || len(values) != 30 || len(values[0].Value) != 48<<10 {
		t.Fatalf("representative page count=%d err=%v", len(values), err)
	}
}
