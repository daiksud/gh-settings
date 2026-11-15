package github

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedRunnerMatching(t *testing.T) {
	modes := []string{"match", "fold order", "empty selected", "unknown IDs", "detail only ID", "list only ID", "mismatch", "read-only collision", "target ID", "owner ID", "target name", "private", "organization", "archive", "visibility", "runner ID", "runner name", "runner OS", "detail changed", "list changed", "ID drift", "ID becomes known", "unknown class", "endpoint disagreement", "endpoint ID disagreement", "missing detail", "missing list", "count mismatch", "target duplicate", "owner duplicate", "target alias", "owner alias", "detail duplicate", "detail alias", "label duplicate", "label alias", "envelope duplicate", "envelope alias", "invalid unicode", "read denied", "pagination"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			targetReads := 0
			writes := 0
			c := runnerFixtureClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method != "GET" {
					writes++
					return aiRepositoryResponse(500, ""), nil
				}
				if req.URL.Path == "/repos/owner/current" {
					targetReads++
					body := strings.Replace(runnerTargetJSON, `"archived":false`, `"archived":true`, 1)
					if targetReads == 2 {
						switch mode {
						case "target ID":
							body = strings.Replace(body, `"id":1`, `"id":9`, 1)
						case "owner ID":
							body = strings.Replace(body, `"id":2`, `"id":9`, 1)
						case "target name":
							body = strings.Replace(body, `owner/current`, `owner/other`, 1)
						case "private":
							body = strings.Replace(body, `"private":false`, `"private":true`, 1)
						case "organization":
							body = strings.Replace(body, `"User"`, `"Organization"`, 1)
						case "archive":
							body = strings.Replace(body, `"archived":true`, `"archived":false`, 1)
						case "visibility":
							body = strings.Replace(body, `"private":false`, `"private":false,"visibility":"private"`, 1)
						}
					}
					switch mode {
					case "target duplicate":
						body = strings.Replace(body, `"id":1`, `"id":1,"id":1`, 1)
					case "owner duplicate":
						body = strings.Replace(body, `"id":2`, `"id":2,"id":2`, 1)
					case "target alias":
						body = strings.Replace(body, `"id":1`, `"id":1,"ID":9`, 1)
					case "owner alias":
						body = strings.Replace(body, `"id":2`, `"id":2,"ID":9`, 1)
					}
					return aiRepositoryResponse(200, body), nil
				}
				if mode == "read denied" && targetReads == 2 {
					return aiRepositoryResponse(403, ""), nil
				}
				detail := !strings.HasSuffix(req.URL.Path, "/labels")
				labels := runnerFixtureLabels("gpu")
				if mode == "empty selected" {
					labels = runnerFixtureLabels("")
				}
				if mode == "fold order" {
					labels = runnerFixtureLabels("GPU")
					labels[0], labels[1] = labels[1], labels[0]
				}
				if mode == "unknown IDs" || mode == "detail only ID" && !detail || mode == "list only ID" && detail || mode == "ID becomes known" && targetReads == 1 {
					for _, v := range labels {
						delete(v, "id")
					}
				}
				if mode == "unknown class" {
					labels[0]["type"] = "future"
				}
				if mode == "endpoint disagreement" && !detail {
					labels[1]["name"] = "other"
				}
				if mode == "endpoint ID disagreement" && !detail {
					labels[1]["id"] = 99
				}
				if targetReads == 2 {
					switch mode {
					case "detail changed":
						if detail {
							labels[1]["name"] = "other"
						}
					case "list changed":
						if !detail {
							labels[1]["name"] = "other"
						}
					case "ID drift":
						labels[1]["id"] = 99
					}
				}
				body := runnerEnvelope(labels)
				if detail {
					v := runnerFixtureRaw(23, labels)
					if targetReads == 2 {
						switch mode {
						case "runner ID":
							v["id"] = 24
						case "runner name":
							v["name"] = "other"
						case "runner OS":
							v["os"] = "other"
						}
					}
					if mode == "missing detail" {
						delete(v, "busy")
					}
					body = runnerJSON(v)
					switch mode {
					case "detail duplicate":
						body = strings.Replace(body, `"id":23`, `"id":23,"id":23`, 1)
					case "detail alias":
						body = strings.Replace(body, `"id":23`, `"id":23,"ID":24`, 1)
					case "invalid unicode":
						body = strings.Replace(body, `"private-runner-name"`, `"\ud800"`, 1)
					}
				} else {
					switch mode {
					case "count mismatch":
						body = strings.Replace(body, `"total_count":2`, `"total_count":1`, 1)
					case "missing list":
						body = `{"total_count":0}`
					case "envelope duplicate":
						body = strings.Replace(body, `"total_count":2`, `"total_count":2,"total_count":2`, 1)
					case "envelope alias":
						body = strings.Replace(body, `"total_count":2`, `"total_count":2,"TOTAL_COUNT":3`, 1)
					}
				}
				switch mode {
				case "label duplicate":
					body = strings.Replace(body, `"name":"gpu"`, `"name":"gpu","name":"gpu"`, 1)
				case "label alias":
					body = strings.Replace(body, `"name":"gpu"`, `"name":"gpu","NAME":"other"`, 1)
				}
				r := aiRepositoryResponse(200, body)
				if mode == "pagination" {
					r.Header.Set("Link", `<https://example.invalid>; rel="next"`)
				}
				return r, nil
			})
			desired := runnerIntent(23, "gpu")
			if mode == "empty selected" {
				desired = runnerIntent(23)
			}
			if mode == "mismatch" {
				desired = runnerIntent(23, "different")
			}
			if mode == "read-only collision" {
				desired = runnerIntent(23, "SELF-HOSTED")
			}
			err := c.VerifyArchivedMatching(runnerRepo, []settings.RunnerLabels{desired}, settings.ArchiveState{ID: 1, FullName: "Owner/Current", Archived: true})
			success := mode == "match" || mode == "fold order" || mode == "empty selected" || mode == "unknown IDs" || mode == "detail only ID" || mode == "list only ID"
			if (err == nil) != success || writes != 0 {
				t.Fatal(fmt.Sprintf("mode=%s err=%v writes=%d", mode, err, writes))
			}
		})
	}
}
