package github

import (
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"strings"
	"testing"
)

func TestPrunedLabelDeletionRequiresExactPreflightAndCompleteAbsence(t *testing.T) {
	const body = `{"id":7,"name":"old","color":"abcdef","description":""}`
	change, _ := settings.NewLabelDelete(settings.LabelState{ID: 7, Name: "old", Color: "abcdef"})
	for _, mode := range []string{"success", "replacement", "value drift", "incomplete", "preflight denied", "delete denied", "unexpected status", "retained", "readback denied", "null readback", "page two denied"} {
		t.Run(mode, func(t *testing.T) {
			deletes := 0
			client := writeTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == "DELETE" {
					deletes++
					if mode == "delete denied" {
						return labelsResponse(r, 403, `{}`), nil
					}
					if mode == "unexpected status" {
						return labelsResponse(r, 200, `{}`), nil
					}
					return labelsResponse(r, 204, ""), nil
				}
				if strings.HasSuffix(r.URL.Path, "/old") {
					value := body
					switch mode {
					case "replacement":
						value = strings.Replace(value, `"id":7`, `"id":8`, 1)
					case "value drift":
						value = strings.Replace(value, "abcdef", "123456", 1)
					case "incomplete":
						value = `{"id":7}`
					case "preflight denied":
						return labelsResponse(r, 403, `{}`), nil
					}
					return labelsResponse(r, 200, value), nil
				}
				if mode == "readback denied" {
					return labelsResponse(r, 403, `{}`), nil
				}
				if mode == "null readback" {
					return labelsResponse(r, 200, `null`), nil
				}
				if mode == "retained" {
					return labelsResponse(r, 200, "["+body+"]"), nil
				}
				if mode == "page two denied" {
					if r.URL.Query().Get("page") == "2" {
						return labelsResponse(r, 403, `{}`), nil
					}
					items := []string{}
					for i := 0; i < 100; i++ {
						items = append(items, fmt.Sprintf(`{"id":%d,"name":"other%d","color":"abcdef","description":""}`, i+100, i))
					}
					return labelsResponse(r, 200, "["+strings.Join(items, ",")+"]"), nil
				}
				return labelsResponse(r, 200, `[]`), nil
			}))
			err := client.DeleteLabel(repository.Repository{Owner: "owner", Name: "current"}, change)
			if (err == nil) != (mode == "success") {
				t.Fatal(mode, err)
			}
			pre := mode == "replacement" || mode == "value drift" || mode == "incomplete" || mode == "preflight denied"
			if pre && deletes != 0 || !pre && deletes != 1 {
				t.Fatal(mode, deletes, err)
			}
		})
	}
}
