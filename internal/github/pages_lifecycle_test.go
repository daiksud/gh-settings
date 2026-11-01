package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestPagesLifecycleUIAndExport(t *testing.T) {
	p := func(s string) *string { return &s }
	source := uimodel.DeployFromABranch
	ui := uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: &source, Branch: p("none")}}
	d, err := PagesDeclaration(ui)
	if err != nil || !d.Delete() {
		t.Fatal(d, err)
	}
	view, err := PagesObservation(settings.NewAbsentPagesObservation())
	if err != nil || !reflect.DeepEqual(view.Pages, ui) {
		t.Fatal(view, err)
	}
	for _, branch := range []string{"None", "NONE", "nOne"} {
		ui.BuildAndDeployment.Branch = p(branch)
		folder := uimodel.PagesRoot
		ui.BuildAndDeployment.SelectFolder = &folder
		d, err := PagesDeclaration(ui)
		if err != nil || d.Delete() || d.Source.Branch != branch {
			t.Fatal(branch, d, err)
		}
	}
}

func TestPagesLifecycleHTTP(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, mode := range []string{"legacy", "workflow", "delete"} {
		for _, failure := range []string{"", "absent", "deny", "401", "409", "422", "redirect", "unexpected", "timeout", "readback404", "readback403", "readback500", "readbackTimeout", "malformed", "mismatch", "identity", "archived", "concurrent404", "concurrentPresent", "optional422", "optionalPreflight", "optionalNoopFinal", "optionalIdentity", "optionalVisibility"} {
			t.Run(mode+"/"+failure, func(t *testing.T) {
				if failure == "absent" && mode != "delete" || (failure == "concurrent404" || failure == "concurrentPresent") && mode != "delete" || (failure == "optional422" || failure == "optionalPreflight" || failure == "optionalNoopFinal" || failure == "optionalIdentity" || failure == "optionalVisibility") && mode == "delete" {
					t.Skip("not applicable")
				}
				p := func(s string) *string { return &s }
				d := settings.PagesDeclaration{BuildType: p(mode)}
				if mode == "legacy" {
					d.Source = &settings.PagesSource{Branch: "Main", Path: "/docs"}
				}
				if mode == "delete" {
					d.BuildType = p("legacy")
					d.Source = &settings.PagesSource{Branch: "none"}
				}
				optional := failure == "optional422" || failure == "optionalPreflight" || failure == "optionalNoopFinal" || failure == "optionalIdentity" || failure == "optionalVisibility"
				if optional {
					d.CNAME = &settings.PagesDomain{Value: p("new.example")}
					if failure == "optionalNoopFinal" {
						d.CNAME = &settings.PagesDomain{}
					}
				}
				present := mode == "delete" && failure != "absent"
				writes, reads, identities := 0, 0, 0
				state := map[string]any{"build_type": "workflow", "cname": nil, "https_enforced": true, "public": true}
				client := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == "/repos/owner/current" {
						identities++
						id := 123
						if failure == "optionalIdentity" && writes > 0 && identities >= 4 {
							id = 456
						}
						if failure == "identity" && identities > 1 {
							id = 456
						}
						if failure == "optionalPreflight" && writes > 0 && identities >= 4 {
							return actionsResponse(r, 403, `{}`), nil
						}
						return actionsResponse(r, 200, fmt.Sprintf(`{"id":%d,"full_name":"owner/current","archived":%t}`, id, failure == "archived")), nil
					}
					if r.URL.Path != "/repos/owner/current/pages" {
						t.Fatal(r.URL)
					}
					if r.Method == "GET" {
						reads++
						if failure == "optionalNoopFinal" && reads >= 4 {
							return actionsResponse(r, 500, `{}`), nil
						}
						if writes > 0 {
							if failure == "readback403" {
								return actionsResponse(r, 403, `{}`), nil
							}
							if failure == "readback500" {
								return actionsResponse(r, 500, `{}`), nil
							}
							if failure == "readbackTimeout" {
								return nil, fmt.Errorf("readback timeout")
							}
							if failure == "readback404" {
								return actionsResponse(r, 404, `{}`), nil
							}
							if failure == "malformed" {
								return actionsResponse(r, 200, `{"cname":null}`), nil
							}
						}
						if !present {
							return actionsResponse(r, 404, `{}`), nil
						}
						body, _ := json.Marshal(state)
						return actionsResponse(r, 200, string(body)), nil
					}
					writes++
					method := "POST"
					if mode == "delete" {
						method = "DELETE"
					}
					if writes == 2 {
						method = "PUT"
					}
					if r.Method != method {
						t.Fatal("unexpected write", r.Method)
					}
					var body []byte
					if r.Body != nil {
						body, _ = io.ReadAll(r.Body)
					}
					if method == "POST" {
						expected := `{"build_type":"workflow"}`
						if mode == "legacy" {
							expected = `{"build_type":"legacy","source":{"branch":"Main","path":"/docs"}}`
						}
						if string(body) != expected {
							t.Fatal("creation payload", string(body))
						}
					} else if method == "DELETE" && len(body) != 0 {
						t.Fatal("delete body", string(body))
					}
					if failure == "timeout" {
						return nil, fmt.Errorf("timeout")
					}
					if failure == "401" {
						return actionsResponse(r, 401, `{}`), nil
					}
					if failure == "409" {
						return actionsResponse(r, 409, `{}`), nil
					}
					if failure == "422" {
						return actionsResponse(r, 422, `{}`), nil
					}
					if failure == "redirect" {
						response := actionsResponse(r, 307, `{}`)
						response.Header.Set("Location", "https://other.invalid/pages")
						return response, nil
					}
					if failure == "deny" {
						return actionsResponse(r, 403, `{}`), nil
					}
					if failure == "unexpected" {
						return actionsResponse(r, 200, `{}`), nil
					}
					if writes == 2 && failure == "optional422" {
						return actionsResponse(r, 422, `{}`), nil
					}
					if failure != "mismatch" && failure != "concurrentPresent" {
						present = method != "DELETE"
						if method == "POST" {
							state["build_type"] = mode
							if mode == "legacy" {
								state["source"] = map[string]any{"branch": "Main", "path": "/docs"}
							}
						}
						if method == "PUT" {
							if string(body) != `{"cname":"new.example"}` {
								t.Fatal(string(body))
							}
							state["cname"] = "new.example"
							if failure == "optionalVisibility" {
								state["public"] = false
							}
						}
					} else if method == "POST" {
						present = true
						state["build_type"] = "legacy"
						state["source"] = map[string]any{"branch": "wrong", "path": "/"}
					}
					status := 204
					if method == "POST" {
						status = 201
					}
					if failure == "concurrent404" || failure == "concurrentPresent" {
						status = 404
					}
					return actionsResponse(r, status, `{}`), nil
				}).rest}
				wrote, err := client.Update(repo, d)
				success := failure == "" || failure == "absent" || failure == "concurrent404" || mode == "delete" && failure == "readback404"
				if (err == nil) != success {
					t.Fatal(wrote, err, writes, reads)
				}
				if optional {
					if failure == "optionalNoopFinal" {
						var pre *pagesPreWriteFailure
						if !errors.As(err, &pre) || writes != 1 {
							t.Fatal("read-only follow-up misclassified", writes, err)
						}
					}
					var created interface{ PagesCreationCompleted() }
					if !errors.As(err, &created) || !wrote {
						t.Fatal("lost confirmed creation", wrote, err)
					}
				}
				if failure == "identity" || failure == "archived" || failure == "absent" {
					if writes != 0 {
						t.Fatal("preflight wrote", writes)
					}
				}
				if writes > 1 && !optional {
					t.Fatal("automatic retry", writes)
				}
				if success && failure != "absent" {
					before := writes
					if _, err := client.Update(repo, d); err != nil || writes != before {
						t.Fatal("not repeatable", err, writes, before)
					}
				}
			})
		}
	}
}

func TestPagesReadEndpointSpecificAbsenceAndUnexpectedSuccess(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, status := range []int{200, 201, 202, 204, 404} {
		client := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) { return actionsResponse(r, status, pagesFixture), nil }).rest}
		observed, err := client.Read(repo)
		if (err == nil) != (status == 200 || status == 404) {
			t.Fatal(status, err)
		}
		if err == nil {
			absent, err := observed.Absent()
			if err != nil || absent != (status == 404) {
				t.Fatal(status, absent, err)
			}
		}
	}
	// The reserved literal branch is acquired without rewriting case or pretending
	// the site is absent, but cannot be exported as valid publishable YAML.
	observed, err := settings.NewPagesObservation(settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "none", Path: "/"}})
	if err != nil {
		t.Fatal(err)
	}
	if absent, _ := observed.Absent(); absent {
		t.Fatal("literal reserved branch inferred absence")
	}
	if _, err := PagesObservation(observed); err == nil {
		t.Fatal("exported contradictory reserved source")
	}
}

func TestPagesReadRejectsMalformedSuccessfulBody(t *testing.T) {
	for _, body := range []string{pagesFixture + ` {}`, pagesFixture + ` trailing`, pagesFixture + ` null`, `{"build_type":`, `[]`, `null`} {
		client := &PagesClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) { return actionsResponse(r, 200, body), nil }).rest}
		observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
		if err == nil {
			t.Fatal("malformed body acquired", body)
		}
		if _, err := observed.Absent(); err == nil {
			t.Fatal("failed read acquired state")
		}
	}
}
