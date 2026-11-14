package github

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// An ambiguous response must not establish identity or selected policy, even
// if the final occurrence matches the desired value and both reads would agree.
func TestEnvironmentInventoryAndDetailRejectAmbiguousJSON(t *testing.T) {
	const inventory = `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`
	const detail = `{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":null,"can_admins_bypass":false}`
	tests := []struct {
		name, inventory, detail string
	}{
		{"inventory duplicate count", `{"total_count":2,"total_count":1,"environments":[{"id":1,"name":"production"}]}`, detail},
		{"inventory duplicate collection", `{"total_count":1,"environments":[],"environments":[{"id":1,"name":"production"}]}`, detail},
		{"inventory duplicate identity", `{"total_count":1,"environments":[{"id":2,"id":1,"name":"production"}]}`, detail},
		{"inventory duplicate name", `{"total_count":1,"environments":[{"id":1,"name":"other","name":"production"}]}`, detail},
		{"inventory count alias", `{"total_count":1,"TOTAL_COUNT":2,"environments":[{"id":1,"name":"production"}]}`, detail},
		{"inventory collection alias", `{"total_count":1,"environments":[{"id":1,"name":"production"}],"ENVIRONMENTS":[]}`, detail},
		{"inventory identity alias", `{"total_count":1,"environments":[{"id":1,"ID":2,"name":"production"}]}`, detail},
		{"inventory name alias", `{"total_count":1,"environments":[{"id":1,"name":"production","NAME":"other"}]}`, detail},
		{"detail duplicate identity", inventory, `{"id":2,"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":null,"can_admins_bypass":false}`},
		{"detail duplicate bypass", inventory, `{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":null,"can_admins_bypass":true,"can_admins_bypass":false}`},
		{"nested duplicate timer", inventory, `{"id":1,"name":"production","protection_rules":[{"type":"wait_timer","wait_timer":60,"wait_timer":0}],"deployment_branch_policy":null,"can_admins_bypass":false}`},
		{"nested duplicate branch", inventory, `{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":{"protected_branches":true,"protected_branches":false,"custom_branch_policies":false},"can_admins_bypass":false}`},
		{"nested reviewer id alias", inventory, `{"id":1,"name":"production","protection_rules":[{"type":"required_reviewers","prevent_self_review":false,"reviewers":[{"type":"User","reviewer":{"id":1,"ID":2,"type":"User"}}]}],"deployment_branch_policy":null,"can_admins_bypass":false}`},
		{"nested reviewer type alias", inventory, `{"id":1,"name":"production","protection_rules":[{"type":"required_reviewers","prevent_self_review":false,"reviewers":[{"type":"User","reviewer":{"id":1,"type":"User","Type":"Organization"}}]}],"deployment_branch_policy":null,"can_admins_bypass":false}`},
		{"trailing inventory", inventory + ` {}`, detail},
		{"trailing detail", inventory, detail + ` {}`},
		{"oversized inventory", `{"total_count":1,"environments":[{"id":1,"name":"production"}],"padding":"` + strings.Repeat("x", environmentResponseLimit+1) + `"}`, detail},
		{"oversized detail", inventory, `{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":null,"can_admins_bypass":false,"url":"` + strings.Repeat("x", environmentResponseLimit+1) + `"}`},
	}
	for _, tc := range tests {
		for _, corruptRead := range []int{1, 2} {
			t.Run(tc.name+"/read_"+string(rune('0'+corruptRead)), func(t *testing.T) {
				reads, writes := 0, 0
				c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.Method != http.MethodGet {
						writes++
						return actionsResponse(r, 400, `{}`), nil
					}
					switch r.URL.Path {
					case "/repos/owner/current/environments":
						reads++
						if reads == corruptRead {
							return actionsResponse(r, 200, tc.inventory), nil
						}
						return actionsResponse(r, 200, inventory), nil
					case "/repos/owner/current/environments/production":
						if reads == corruptRead {
							return actionsResponse(r, 200, tc.detail), nil
						}
						return actionsResponse(r, 200, detail), nil
					case "/repos/owner/current/environments/production/deployment_protection_rules":
						return actionsResponse(r, 200, customEnvironmentEmpty), nil
					default:
						t.Fatalf("unexpected request %s", r.URL)
						return nil, nil
					}
				}).rest}
				repo := repository.Repository{Owner: "owner", Name: "current"}
				for i := 1; i <= corruptRead; i++ {
					observation, err := c.Read(repo, []string{"production"})
					if (err != nil) != (i == corruptRead) {
						t.Fatalf("read %d: %v", i, err)
					}
					if i == corruptRead {
						if _, err := observation.Values(); err == nil {
							t.Fatal("ambiguous response became a complete observation")
						}
					}
				}
				if writes != 0 {
					t.Fatalf("observation attempted %d writes", writes)
				}
			})
		}
	}
}

func TestEnvironmentUpdateRejectsAmbiguousAcknowledgment(t *testing.T) {
	for _, body := range []string{
		`{"id":2,"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":null,"can_admins_bypass":true}`,
		`{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":null,"can_admins_bypass":false,"can_admins_bypass":true}`,
		`{"id":1,"name":"production","protection_rules":[{"type":"wait_timer","wait_timer":30,"wait_timer":0}],"deployment_branch_policy":null,"can_admins_bypass":true}`,
		`{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":null,"can_admins_bypass":true} {}`,
	} {
		t.Run(body, func(t *testing.T) {
			writes, detailReads := 0, 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodPut {
					writes++
					return actionsResponse(r, 200, body), nil
				}
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected method %s", r.Method)
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
				case "/repos/owner/current/environments":
					return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case "/repos/owner/current/environments/production":
					detailReads++
					return actionsResponse(r, 200, emptyEnvironmentFixture), nil
				case "/repos/owner/current/environments/production/deployment_protection_rules":
					return actionsResponse(r, 200, customEnvironmentEmpty), nil
				default:
					t.Fatalf("unexpected request %s", r.URL)
					return nil, nil
				}
			}).rest}
			yes := true
			changed, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, settings.EnvironmentDeclaration{Name: "production", CanAdminsBypass: &yes})
			var beforeWrite interface{ ReadFailedBeforeWrite() }
			if err == nil || changed || writes != 1 || detailReads != 2 || errors.As(err, &beforeWrite) || !strings.Contains(err.Error(), "write may have applied") {
				t.Fatalf("changed=%v err=%v writes=%d detail reads=%d", changed, err, writes, detailReads)
			}
		})
	}
}

func TestEnvironmentReadKeepsUnownedInventoryAndReviewerMetadata(t *testing.T) {
	c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method %s", r.Method)
		}
		switch r.URL.Path {
		case "/repos/owner/current/environments":
			return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production","node_id":"node","extra_metadata":true}],"other_metadata":true}`), nil
		case "/repos/owner/current/environments/production":
			return actionsResponse(r, 200, `{"id":1,"name":"production","url":"https://example.invalid","protection_rules":[{"type":"required_reviewers","prevent_self_review":false,"reviewers":[{"type":"User","reviewer":{"id":2,"type":"User","login":"reviewer","avatar_url":"https://example.invalid"}}]}],"deployment_branch_policy":null,"can_admins_bypass":false}`), nil
		case "/repos/owner/current/environments/production/deployment_protection_rules":
			return actionsResponse(r, 200, customEnvironmentEmpty), nil
		default:
			t.Fatalf("unexpected request %s", r.URL)
			return nil, nil
		}
	}).rest}
	observation, err := c.Read(repository.Repository{Owner: "owner", Name: "current"}, []string{"production"})
	if err != nil {
		t.Fatal(err)
	}
	values, err := observation.Values()
	if err != nil || len(values) != 1 || values[0].ID != 1 || values[0].Policy.Reviewers == nil || len(*values[0].Policy.Reviewers) != 1 || (*values[0].Policy.Reviewers)[0].ID != 2 {
		t.Fatalf("incomplete observation: %v", err)
	}
}

// A syntactically complete body on a partial-content response cannot prove a
// complete inventory or detail observation, including on the second read.
func TestEnvironmentReadRejectsPartialContent(t *testing.T) {
	for _, endpoint := range []string{"inventory", "detail"} {
		for _, corruptRead := range []int{1, 2} {
			t.Run(endpoint+"/read_"+string(rune('0'+corruptRead)), func(t *testing.T) {
				reads, writes := 0, 0
				c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.Method != http.MethodGet {
						writes++
						return actionsResponse(r, 400, `{}`), nil
					}
					switch r.URL.Path {
					case "/repos/owner/current/environments":
						reads++
						status := http.StatusOK
						if endpoint == "inventory" && reads == corruptRead {
							status = http.StatusPartialContent
						}
						return actionsResponse(r, status, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
					case "/repos/owner/current/environments/production":
						status := http.StatusOK
						if endpoint == "detail" && reads == corruptRead {
							status = http.StatusPartialContent
						}
						return actionsResponse(r, status, emptyEnvironmentFixture), nil
					case "/repos/owner/current/environments/production/deployment_protection_rules":
						return actionsResponse(r, 200, customEnvironmentEmpty), nil
					default:
						t.Fatalf("unexpected request %s", r.URL)
						return nil, nil
					}
				}).rest}
				repo := repository.Repository{Owner: "owner", Name: "current"}
				for i := 1; i <= corruptRead; i++ {
					observed, err := c.Read(repo, []string{"production"})
					if (err != nil) != (i == corruptRead) {
						t.Fatalf("read %d: %v", i, err)
					}
					if i == corruptRead {
						if _, err := observed.Values(); err == nil {
							t.Fatal("partial response became a complete observation")
						}
					}
				}
				if writes != 0 {
					t.Fatalf("observation attempted %d writes", writes)
				}
			})
		}
	}
}

func TestEnvironmentUpdateRejectsPartialContent(t *testing.T) {
	for _, endpoint := range []string{"inventory", "detail", "acknowledgment"} {
		t.Run(endpoint, func(t *testing.T) {
			writes, detailReads := 0, 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodPut {
					writes++
					status := http.StatusOK
					if endpoint == "acknowledgment" {
						status = http.StatusPartialContent
					}
					return actionsResponse(r, status, `{"id":1,"name":"production","protection_rules":[],"deployment_branch_policy":null,"can_admins_bypass":true}`), nil
				}
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected method %s", r.Method)
				}
				switch r.URL.Path {
				case "/repos/owner/current":
					return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
				case "/repos/owner/current/environments":
					status := http.StatusOK
					if endpoint == "inventory" {
						status = http.StatusPartialContent
					}
					return actionsResponse(r, status, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
				case "/repos/owner/current/environments/production":
					detailReads++
					status := http.StatusOK
					if endpoint == "detail" {
						status = http.StatusPartialContent
					}
					return actionsResponse(r, status, emptyEnvironmentFixture), nil
				case "/repos/owner/current/environments/production/deployment_protection_rules":
					return actionsResponse(r, 200, customEnvironmentEmpty), nil
				default:
					t.Fatalf("unexpected request %s", r.URL)
					return nil, nil
				}
			}).rest}
			yes := true
			changed, err := c.Update(repository.Repository{Owner: "owner", Name: "current"}, settings.EnvironmentDeclaration{Name: "production", CanAdminsBypass: &yes})
			var beforeWrite interface{ ReadFailedBeforeWrite() }
			if err == nil || changed {
				t.Fatalf("changed=%v err=%v", changed, err)
			}
			if endpoint == "acknowledgment" {
				if writes != 1 || detailReads != 2 || errors.As(err, &beforeWrite) || !strings.Contains(err.Error(), "write may have applied") {
					t.Fatalf("err=%v writes=%d detail reads=%d", err, writes, detailReads)
				}
			} else if writes != 0 || !errors.As(err, &beforeWrite) {
				t.Fatalf("err=%v writes=%d", err, writes)
			}
		})
	}
}
