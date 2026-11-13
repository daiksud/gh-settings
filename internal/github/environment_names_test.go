package github

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestEnvironmentNamesIndependentCompleteInventory(t *testing.T) {
	for _, mode := range []string{"complete", "late-error", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			pages := 0
			c := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" || r.URL.Path != "/repos/owner/current/environments" {
					t.Fatal("names acquired policy or wrote", r.Method, r.URL)
				}
				pages++
				if pages == 2 && mode == "late-error" {
					return actionsResponse(r, 403, `{}`), nil
				}
				body := `{"total_count":101,"environments":[`
				if pages == 1 {
					for i := range 100 {
						if i > 0 {
							body += ","
						}
						body += fmt.Sprintf(`{"id":%d,"name":"env-%03d"}`, i+1, 99-i)
					}
				} else {
					if mode == "duplicate" {
						body += `{"id":1,"name":"env-099"}`
					} else {
						body += `{"id":101,"name":"last"}`
					}
				}
				body += `]}`
				return actionsResponse(r, 200, body), nil
			}).rest}
			names, err := c.Names(repository.Repository{Owner: "owner", Name: "current"})
			if mode != "complete" {
				if err == nil || names != nil {
					t.Fatal("partial names became complete", len(names), err)
				}
				return
			}
			if err != nil || pages != 2 || len(names) != 101 || !reflect.DeepEqual(names[:2], []string{"env-000", "env-001"}) {
				t.Fatal(len(names), pages, err)
			}
		})
	}
}
