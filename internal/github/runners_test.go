package github

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

var runnerRepo = repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}

const runnerTargetJSON = `{"id":1,"full_name":"owner/current","private":false,"archived":false,"owner":{"id":2,"login":"owner","type":"User"}}`

func runnerIntent(id int64, names ...string) settings.RunnerLabels {
	if names == nil {
		names = []string{}
	}
	return settings.RunnerLabels{ID: id, CustomLabels: &names}
}
func runnerFixtureLabels(custom string) []map[string]any {
	v := []map[string]any{{"id": 1, "name": "self-hosted", "type": "read-only"}}
	if custom != "" {
		v = append(v, map[string]any{"id": 2, "name": custom, "type": "custom"})
	}
	return v
}
func runnerFixtureRaw(id int64, labels []map[string]any) map[string]any {
	return map[string]any{"id": id, "name": "private-runner-name", "os": "private-os", "status": "online", "busy": false, "labels": labels}
}
func runnerJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
func runnerEnvelope(v []map[string]any) string {
	return runnerJSON(map[string]any{"total_count": len(v), "labels": v})
}
func runnerFixtureClient(t *testing.T, fn func(*http.Request) (*http.Response, error)) *RunnersClient {
	return &RunnersClient{actionsClient(t, fn).rest}
}
func TestRunnerLabelsCompleteRead(t *testing.T) {
	cases := []string{"valid", "optional IDs absent", "only detail ID", "only list ID", "unknown type", "null ID", "zero ID", "missing labels", "missing busy", "missing status", "wrong ID", "label count", "label disagreement", "ID disagreement", "cross ID collision", "duplicate name", "unknown status", "trailing JSON", "pagination", "target wrong", "denied"}
	for _, mode := range cases {
		t.Run(mode, func(t *testing.T) {
			c := runnerFixtureClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatal(r.Method)
				}
				if r.URL.Path == "/repos/owner/current" {
					body := runnerTargetJSON
					if mode == "target wrong" {
						body = strings.Replace(body, `"User"`, `"Organization"`, 1)
					}
					return aiRepositoryResponse(200, body), nil
				}
				labels := runnerFixtureLabels("gpu")
				detail := !strings.HasSuffix(r.URL.Path, "/labels")
				if mode == "optional IDs absent" || (mode == "only detail ID" && !detail) || (mode == "only list ID" && detail) {
					delete(labels[0], "id")
				}
				switch mode {
				case "unknown type":
					delete(labels[0], "type")
				case "null ID":
					labels[0]["id"] = nil
				case "zero ID":
					labels[0]["id"] = 0
				case "duplicate name":
					labels[1]["name"] = "SELF-HOSTED"
				case "ID disagreement":
					if !detail {
						labels[0]["id"] = 9
					}
				case "label disagreement":
					if !detail {
						labels[1]["name"] = "other"
					}
				case "cross ID collision":
					if detail {
						delete(labels[1], "id")
					} else {
						delete(labels[0], "id")
						labels[1]["id"] = 1
					}
				}
				body := runnerEnvelope(labels)
				if detail {
					v := runnerFixtureRaw(23, labels)
					switch mode {
					case "missing labels":
						delete(v, "labels")
					case "missing busy":
						delete(v, "busy")
					case "missing status":
						delete(v, "status")
					case "wrong ID":
						v["id"] = 24
					case "unknown status":
						v["status"] = "future"
					}
					body = runnerJSON(v)
				} else if mode == "label count" {
					body = strings.Replace(body, `"total_count":2`, `"total_count":1`, 1)
				}
				if mode == "trailing JSON" {
					body += " {}"
				}
				status := 200
				if mode == "denied" {
					status = 403
				}
				resp := aiRepositoryResponse(status, body)
				if mode == "pagination" {
					resp.Header.Set("Link", `<https://api.github.com/else>; rel="next"`)
				}
				return resp, nil
			})
			got, e := c.ReadSelected(runnerRepo, []settings.RunnerLabels{runnerIntent(23, "gpu")})
			valid := mode == "valid" || mode == "unknown status" || mode == "optional IDs absent" || mode == "only detail ID" || mode == "only list ID"
			if (e == nil) != valid {
				t.Fatal(mode, e)
			}
			if valid {
				v, e := got.Values()
				if e != nil || len(v) != 1 || !settings.RunnerLabelsMatch(v[0], runnerIntent(23, "gpu")) {
					t.Fatal(v, e)
				}
			}
		})
	}
}
func TestRunnerLabelsFreshConditionalWriteAndReadback(t *testing.T) {
	modes := []string{"changed", "clear", "matching", "case match", "zero readonly", "no IDs", "detail-only ID", "list-only ID", "detail ID ack lost", "list ID ack lost", "custom ID changed", "busy offline", "readonly collision", "archived", "pre denied", "put denied", "put network", "ack status", "ack null", "ack count", "ack extra", "ack reclassified", "ack ID lost", "ack ID drift", "ack readonly spelling", "readback denied", "readback extra", "readback ID lost", "readback ID drift", "readback readonly added", "readback name", "readback os", "readback runner", "readback repo", "readback owner"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			writes, details, lists := 0, 0, 0
			custom := "old"
			desired := runnerIntent(23, "gpu")
			if mode == "clear" {
				desired = runnerIntent(23)
			}
			if mode == "matching" {
				custom = "gpu"
			}
			if mode == "case match" {
				custom = "GPU"
			}
			if mode == "readonly collision" {
				desired = runnerIntent(23, "SELF-HOSTED")
			}
			c := runnerFixtureClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					body := runnerTargetJSON
					if mode == "archived" {
						body = strings.Replace(body, `"archived":false`, `"archived":true`, 1)
					}
					if writes > 0 && mode == "readback repo" {
						body = strings.Replace(body, `"id":1`, `"id":9`, 1)
					}
					if writes > 0 && mode == "readback owner" {
						body = strings.Replace(body, `"id":2`, `"id":9`, 1)
					}
					return aiRepositoryResponse(200, body), nil
				}
				if !strings.HasPrefix(r.URL.Path, "/repos/owner/current/actions/runners/23") {
					t.Fatal("unexpected endpoint", r.URL)
				}
				detail := !strings.HasSuffix(r.URL.Path, "/labels")
				ack := r.Method == "PUT"
				if ack {
					writes++
					var p map[string]any
					b, _ := io.ReadAll(r.Body)
					json.Unmarshal(b, &p)
					want := map[string]any{"labels": *desired.CustomLabels}
					wb, _ := json.Marshal(want)
					if string(b) != string(wb) {
						t.Fatal("payload", string(b))
					}
					custom = "gpu"
					if mode == "clear" {
						custom = ""
					}
					if mode == "put network" {
						return nil, errors.New("private upstream")
					}
				}
				if r.Method != "GET" && !ack {
					t.Fatal("forbidden method", r.Method)
				}
				if !ack {
					if detail {
						details++
					} else {
						lists++
					}
				}
				labels := runnerFixtureLabels(custom)
				if mode == "zero readonly" {
					labels = labels[1:]
				}
				if mode == "no IDs" || ((mode == "detail-only ID" || mode == "detail ID ack lost") && !detail && writes == 0) || ((mode == "list-only ID" || mode == "list ID ack lost") && detail && writes == 0) {
					if len(labels) > 0 {
						delete(labels[0], "id")
					}
				}
				if (ack && (mode == "ack ID lost" || mode == "detail ID ack lost" || mode == "list ID ack lost")) || (!ack && writes > 0 && mode == "readback ID lost") {
					delete(labels[0], "id")
				}
				if (ack && mode == "ack ID drift") || (!ack && writes > 0 && mode == "readback ID drift") {
					labels[0]["id"] = 9
				}
				if (ack && mode == "ack extra") || (!ack && writes > 0 && mode == "readback extra") {
					labels = append(labels, map[string]any{"name": "extra", "type": "custom"})
				}
				if !ack && writes > 0 && mode == "readback readonly added" {
					labels = append(labels, map[string]any{"name": "new-readonly", "type": "read-only"})
				}
				if ack && mode == "ack reclassified" {
					labels[0]["type"] = "custom"
				}
				if ack && mode == "ack readonly spelling" {
					labels[0]["name"] = "SELF-HOSTED"
				}
				if writes > 0 && mode == "custom ID changed" {
					labels[1]["id"] = 99
				}
				body := runnerEnvelope(labels)
				status := 200
				if detail {
					v := runnerFixtureRaw(23, labels)
					if writes > 0 {
						switch mode {
						case "busy offline":
							v["busy"], v["status"] = true, "offline"
						case "readback name":
							v["name"] = "changed"
						case "readback os":
							v["os"] = "changed"
						case "readback runner":
							v["id"] = 24
						}
					}
					body = runnerJSON(v)
				}
				if ack {
					switch mode {
					case "ack status":
						status = 201
					case "ack null":
						body = "null"
					case "ack count":
						body = strings.Replace(body, `"total_count":2`, `"total_count":1`, 1)
					case "put denied":
						status = 403
					}
				}
				if mode == "pre denied" || (writes > 0 && !ack && mode == "readback denied") {
					status = 403
					body = `{"message":"private upstream"}`
				}
				return aiRepositoryResponse(status, body), nil
			})
			wrote, e := c.Update(runnerRepo, desired)
			success := mode == "custom ID changed" || mode == "changed" || mode == "clear" || mode == "matching" || mode == "case match" || mode == "zero readonly" || mode == "no IDs" || mode == "detail-only ID" || mode == "list-only ID" || mode == "busy offline"
			if (e == nil) != success {
				t.Fatal(mode, wrote, e)
			}
			noop := mode == "matching" || mode == "case match"
			pre := mode == "readonly collision" || mode == "archived" || mode == "pre denied"
			if writes > 1 || (noop || pre) && writes != 0 || success && !noop && (writes != 1 || !wrote || details != 2 || lists != 2) {
				t.Fatal(writes, details, lists, wrote, e)
			}
			if e != nil {
				var marker interface{ ReadFailedBeforeWrite() }
				if errors.As(e, &marker) != pre || strings.Contains(e.Error(), "private") {
					t.Fatal("failure certainty/privacy", e)
				}
			}
		})
	}
}
func TestRunnerInventoryPaginationAndSelectionIsolation(t *testing.T) {
	for _, mode := range []string{"empty", "paged", "missing total", "changing total", "duplicate ID", "wrong total", "unsafe next", "short next", "detail denied", "label unknown"} {
		t.Run(mode, func(t *testing.T) {
			detailReads := 0
			c := runnerFixtureClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					return aiRepositoryResponse(200, runnerTargetJSON), nil
				}
				if r.URL.Path == "/repos/owner/current/actions/runners" {
					page := r.URL.Query().Get("page")
					rows := []any{}
					total := 101
					if mode == "empty" {
						total = 0
					} else if page == "1" {
						for id := int64(1); id <= 100; id++ {
							rows = append(rows, runnerFixtureRaw(id, runnerFixtureLabels("gpu")))
						}
					} else {
						rows = append(rows, runnerFixtureRaw(101, runnerFixtureLabels("gpu")))
					}
					if mode == "duplicate ID" && page == "2" {
						rows[0] = runnerFixtureRaw(1, runnerFixtureLabels("gpu"))
					}
					if mode == "short next" && page == "1" {
						rows = rows[:1]
					}
					if mode == "changing total" && page == "2" {
						total = 102
					}
					if mode == "wrong total" {
						total = 0
					}
					raw := map[string]any{"total_count": total, "runners": rows}
					if mode == "missing total" {
						delete(raw, "total_count")
					}
					resp := aiRepositoryResponse(200, runnerJSON(raw))
					if page == "1" && mode != "empty" {
						host := "api.github.com"
						if mode == "unsafe next" {
							host = "evil.test"
						}
						resp.Header.Set("Link", fmt.Sprintf(`<https://%s/repos/owner/current/actions/runners?per_page=100&page=2>; rel="next"`, host))
					}
					return resp, nil
				}
				if strings.HasSuffix(r.URL.Path, "/labels") {
					labels := runnerFixtureLabels("gpu")
					if mode == "label unknown" {
						delete(labels[0], "type")
					}
					return aiRepositoryResponse(200, runnerEnvelope(labels)), nil
				}
				var id int64
				fmt.Sscanf(r.URL.Path, "/repos/owner/current/actions/runners/%d", &id)
				detailReads++
				if mode == "detail denied" {
					return aiRepositoryResponse(403, `{}`), nil
				}
				return aiRepositoryResponse(200, runnerJSON(runnerFixtureRaw(id, runnerFixtureLabels("gpu")))), nil
			})
			o, e := c.ReadAll(runnerRepo)
			valid := mode == "empty" || mode == "paged"
			if (e == nil) != valid {
				t.Fatal(mode, e)
			}
			if valid {
				v, e := o.Values()
				want := 0
				if mode == "paged" {
					want = 101
				}
				if e != nil || len(v) != want || detailReads != want {
					t.Fatal(len(v), detailReads, e)
				}
			}
		})
	}
	calls := []string{}
	c := runnerFixtureClient(t, func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.URL.Path)
		if r.URL.Path == "/repos/owner/current" {
			return aiRepositoryResponse(200, runnerTargetJSON), nil
		}
		if strings.HasSuffix(r.URL.Path, "/labels") {
			return aiRepositoryResponse(200, runnerEnvelope(runnerFixtureLabels("gpu"))), nil
		}
		return aiRepositoryResponse(200, runnerJSON(runnerFixtureRaw(23, runnerFixtureLabels("gpu")))), nil
	})
	if _, e := c.ReadSelected(runnerRepo, nil); e != nil || len(calls) != 0 {
		t.Fatal(calls, e)
	}
	if _, e := c.ReadSelected(runnerRepo, []settings.RunnerLabels{runnerIntent(23)}); e != nil || !reflect.DeepEqual(calls, []string{"/repos/owner/current", "/repos/owner/current/actions/runners/23", "/repos/owner/current/actions/runners/23/labels"}) {
		t.Fatal(calls, e)
	}
}
