package github

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// Integration-only evidence for the parent identity and inventory changes.
func TestMixedEnvironmentVariableParentAndPaginatedInventory(t *testing.T) {
	for _, mode := range []string{"complete", "ambiguous second page", "changed parent"} {
		t.Run(mode, func(t *testing.T) {
			parents, pages, writes := 0, 0, 0
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					writes++
					t.Fatalf("unexpected write")
				}
				if r.URL.Path == "/repos/owner/current/environments/prod" {
					parents++
					id := 42
					if mode == "changed parent" && parents == 2 {
						id = 43
					}
					return labelsResponse(r, 200, fmt.Sprintf(`{"id":%d,"name":"prod"}`, id)), nil
				}
				if r.URL.Path != "/repos/owner/current/environments/prod/variables" {
					t.Fatalf("unexpected endpoint %s", r.URL.Path)
				}
				pages++
				if pages == 1 {
					entries := make([]string, 30)
					for i := range entries {
						entries[i] = fmt.Sprintf(`{"name":"ITEM_%d","value":"example"}`, i)
					}
					return labelsResponse(r, 200, `{"total_count":31,"variables":[`+strings.Join(entries, ",")+`]}`), nil
				}
				if mode == "ambiguous second page" {
					return labelsResponse(r, 200, `{"total_count":31,"total_count":0,"variables":[{"name":"LAST","value":"example"}]}`), nil
				}
				return labelsResponse(r, 200, `{"total_count":31,"variables":[{"name":"LAST","value":"example"}]}`), nil
			}))
			inventory, err := client.ReadEnvironmentVariables(repository.Repository{Owner: "owner", Name: "current"}, "prod")
			if mode == "complete" {
				values, valuesErr := inventory.Values()
				id, idErr := inventory.EnvironmentID("prod")
				if err != nil || valuesErr != nil || idErr != nil || id != 42 || len(values) != 31 || parents != 2 || pages != 2 || writes != 0 {
					t.Fatalf("err=%v valuesErr=%v idErr=%v id=%d count=%d parents=%d pages=%d writes=%d", err, valuesErr, idErr, id, len(values), parents, pages, writes)
				}
			} else {
				wantParents := 2
				if mode == "ambiguous second page" {
					wantParents = 1
				}
				if err == nil || parents != wantParents || pages != 2 || writes != 0 {
					t.Fatalf("err=%v parents=%d pages=%d writes=%d", err, parents, pages, writes)
				}
			}
		})
	}
}
