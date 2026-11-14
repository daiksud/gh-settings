package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRunnerPruneCompleteAndFailClosed(t *testing.T) {
	for _, mode := range []string{"valid", "unknown runner", "unknown label", "unknown status", "busy", "missing busy", "null ephemeral", "wrong version type", "list drift", "detail mismatch", "label mismatch", "count", "target"} {
		t.Run(mode, func(t *testing.T) {
			listCalls := 0
			c := runnerFixtureClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatalf("unexpected write %s", r.URL)
				}
				if r.URL.Path == "/repos/owner/current" {
					body := runnerTargetJSON
					if mode == "target" {
						body = strings.Replace(body, `"User"`, `"Organization"`, 1)
					}
					return aiRepositoryResponse(200, body), nil
				}
				labels := runnerFixtureLabels("gpu")
				row := runnerFixtureRaw(23, labels)
				row["ephemeral"] = false
				row["version"] = "2.323.0"
				switch mode {
				case "unknown runner":
					row["new_configuration"] = true
				case "unknown label":
					labels[0]["new"] = true
				case "unknown status":
					row["status"] = "unknown"
				case "busy":
					row["busy"] = true
				case "missing busy":
					delete(row, "busy")
				case "null ephemeral":
					row["ephemeral"] = nil
				case "wrong version type":
					row["version"] = false
				}
				if r.URL.Path == "/repos/owner/current/actions/runners" {
					listCalls++
					if mode == "list drift" && listCalls > 1 {
						row["name"] = "changed"
					}
					total := 1
					if mode == "count" {
						total = 2
					}
					return aiRepositoryResponse(200, runnerJSON(map[string]any{"total_count": total, "runners": []any{row}})), nil
				}
				if strings.HasSuffix(r.URL.Path, "/labels") {
					if mode == "label mismatch" {
						labels[1]["name"] = "other"
					}
					return aiRepositoryResponse(200, runnerEnvelope(labels)), nil
				}
				if mode == "detail mismatch" {
					row["name"] = "other"
				}
				return aiRepositoryResponse(200, runnerJSON(row)), nil
			})
			observed, err := c.ReadPrune(runnerRepo)
			if err == nil {
				_, err = observed.Candidates(nil)
			}
			if (err == nil) != (mode == "valid") {
				t.Fatalf("%s: %v", mode, err)
			}
		})
	}
}
func TestRunnerPruneDeleteRequiresFreshEvidenceAndIndependentAbsence(t *testing.T) {
	for _, mode := range []string{"success", "busy before", "status drift", "name drift", "new runner", "missing before", "204 body", "404", "ack only", "verify denied"} {
		t.Run(mode, func(t *testing.T) {
			removed := false
			deletes := 0
			reads := 0
			c := runnerFixtureClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					return aiRepositoryResponse(200, runnerTargetJSON), nil
				}
				if r.Method == "DELETE" {
					deletes++
					if r.URL.Path != "/repos/owner/current/actions/runners/23" {
						t.Fatal(r.URL)
					}
					switch mode {
					case "404":
						return aiRepositoryResponse(404, ""), nil
					case "204 body":
						return aiRepositoryResponse(204, "unexpected"), nil
					case "ack only":
						return aiRepositoryResponse(204, ""), nil
					default:
						removed = true
						return aiRepositoryResponse(204, ""), nil
					}
				}
				if r.Method != "GET" {
					t.Fatal(r.Method)
				}
				if removed && mode == "verify denied" {
					return aiRepositoryResponse(403, ""), nil
				}
				labels := runnerFixtureLabels("gpu")
				row := runnerFixtureRaw(23, labels)
				if reads > 0 {
					switch mode {
					case "busy before":
						row["busy"] = true
					case "status drift":
						row["status"] = "offline"
					case "name drift":
						row["name"] = "replacement"
					}
				}
				if r.URL.Path == "/repos/owner/current/actions/runners" {
					entries := []any{}
					if !removed && !(reads > 0 && mode == "missing before") {
						entries = append(entries, row)
					}
					if reads > 0 && mode == "new runner" {
						entries = append(entries, runnerFixtureRaw(24, labels))
					}
					return aiRepositoryResponse(200, runnerJSON(map[string]any{"total_count": len(entries), "runners": entries})), nil
				}
				if strings.HasSuffix(r.URL.Path, "/labels") {
					return aiRepositoryResponse(200, runnerEnvelope(labels)), nil
				}
				return aiRepositoryResponse(200, runnerJSON(row)), nil
			})
			initial, err := c.ReadPrune(runnerRepo)
			if err != nil {
				t.Fatal(err)
			}
			reads++
			attempted, err := c.DeletePrune(runnerRepo, initial, []int64{23}, 23)
			if mode == "success" {
				if err != nil || !attempted || deletes != 1 {
					t.Fatal(attempted, deletes, err)
				}
				return
			}
			if err == nil {
				t.Fatal("unexpected success")
			}
			pre := mode == "busy before" || mode == "status drift" || mode == "name drift" || mode == "new runner" || mode == "missing before"
			if pre && (attempted || deletes != 0) {
				t.Fatal("preflight wrote", mode)
			}
			if !pre && (!attempted || deletes != 1) {
				t.Fatal("missing attempted status", fmt.Sprint(attempted, deletes))
			}
		})
	}
}
func TestRunnerPruneRetainedNoiseAndLabelEvidence(t *testing.T) {
	labels := []settings.RunnerPruneLabel{{Name: "self-hosted", Kind: "read-only"}, {Name: "old", Kind: "custom"}}
	a := settings.RunnerPruneObservation{Target: settings.RunnerPruneTarget{ID: 1, OwnerID: 2}, Entries: []settings.RunnerPruneState{{ID: 23, Name: "keep", OS: "linux", Status: "online", Labels: labels}, {ID: 24, Name: "delete", OS: "macos", Status: "offline", Labels: []settings.RunnerPruneLabel{}}}}
	b := a
	b.Entries = append([]settings.RunnerPruneState{}, a.Entries...)
	b.Entries[0].Status = "offline"
	b.Entries[0].Busy = true
	v := "new"
	b.Entries[0].Version = &v
	if err := settings.VerifyRunnerPrune(a, b, []int64{24}, nil); err != nil {
		t.Fatal(err)
	}
	b.Entries[1].Busy = true
	if settings.VerifyRunnerPrune(a, b, []int64{24}, nil) == nil {
		t.Fatal("candidate busy accepted")
	}
}

func TestRunnerPrunePaginationAndCompleteInventory(t *testing.T) {
	for _, mode := range []string{"complete", "missing next", "unsafe next", "changing count", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			c := runnerFixtureClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					return aiRepositoryResponse(200, runnerTargetJSON), nil
				}
				labels := runnerFixtureLabels("")
				if r.URL.Path == "/repos/owner/current/actions/runners" {
					page := r.URL.Query().Get("page")
					rows := []any{}
					start, end := int64(1), int64(100)
					if page == "2" {
						start, end = 101, 101
					}
					for id := start; id <= end; id++ {
						if mode == "duplicate" && id == 101 {
							id = 1
							rows = append(rows, runnerFixtureRaw(id, labels))
							break
						}
						rows = append(rows, runnerFixtureRaw(id, labels))
					}
					total := 101
					if page == "2" && mode == "changing count" {
						total = 102
					}
					response := aiRepositoryResponse(200, runnerJSON(map[string]any{"total_count": total, "runners": rows}))
					if page == "1" && mode != "missing next" {
						next := "https://api.github.com/repos/owner/current/actions/runners?per_page=100&page=2"
						if mode == "unsafe next" {
							next = "https://evil.example/other?page=2&per_page=100"
						}
						response.Header.Set("Link", fmt.Sprintf(`<%s>; rel="next"`, next))
					}
					return response, nil
				}
				if strings.HasSuffix(r.URL.Path, "/labels") {
					return aiRepositoryResponse(200, runnerEnvelope(labels)), nil
				}
				var id int64
				fmt.Sscanf(r.URL.Path, "/repos/owner/current/actions/runners/%d", &id)
				return aiRepositoryResponse(200, runnerJSON(runnerFixtureRaw(id, labels))), nil
			})
			o, err := c.ReadPrune(runnerRepo)
			if mode == "complete" {
				if err != nil || len(o.Entries) != 101 {
					t.Fatal(len(o.Entries), err)
				}
			} else if err == nil {
				t.Fatal("accepted", mode)
			}
		})
	}
}
func TestRunnerPruneOperationalChangesOnlyBlockCandidates(t *testing.T) {
	detailCalls := 0
	c := runnerFixtureClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/repos/owner/current" {
			return aiRepositoryResponse(200, runnerTargetJSON), nil
		}
		labels := runnerFixtureLabels("")
		row := runnerFixtureRaw(23, labels)
		if r.URL.Path == "/repos/owner/current/actions/runners" {
			if detailCalls > 0 {
				row["status"] = "offline"
				row["busy"] = true
			}
			return aiRepositoryResponse(200, runnerJSON(map[string]any{"total_count": 1, "runners": []any{row}})), nil
		}
		if strings.HasSuffix(r.URL.Path, "/labels") {
			return aiRepositoryResponse(200, runnerEnvelope(labels)), nil
		}
		detailCalls++
		return aiRepositoryResponse(200, runnerJSON(row)), nil
	})
	o, err := c.ReadPrune(runnerRepo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = o.Candidates([]settings.RunnerLabels{runnerIntent(23)}); err != nil {
		t.Fatal("retained operational noise", err)
	}
	if _, err = o.Candidates(nil); err == nil {
		t.Fatal("candidate operational drift accepted")
	}
}

func TestRunnerPruneRejectsDuplicateEvidenceMembers(t *testing.T) {
	for _, raw := range []string{`{"id":23,"id":24,"name":"runner","os":"linux","status":"online","busy":false,"labels":[]}`, `{"id":23,"name":"runner","os":"linux","status":"online","busy":false,"labels":[{"name":"x","type":"custom","type":"read-only"}]}`} {
		if _, err := pruneRunner(json.RawMessage(raw)); err == nil {
			t.Fatal("accepted duplicate member", raw)
		}
	}
}

func TestRunnerPruneRejectsHiddenNextPage(t *testing.T) {
	for _, mode := range []string{"unquoted", "multiple relations", "second header", "extended relation"} {
		t.Run(mode, func(t *testing.T) {
			c := runnerFixtureClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected write: %s", r.Method)
				}
				response := aiRepositoryResponse(200, runnerJSON(map[string]any{"total_count": 0, "runners": []any{}}))
				switch mode {
				case "unquoted":
					response.Header.Add("Link", `<https://api.github.com/next>; rel=next`)
				case "multiple relations":
					response.Header.Add("Link", `<https://api.github.com/next>; rel="next last"`)
				case "second header":
					response.Header.Add("Link", `<https://api.github.com/prev>; rel=prev`)
					response.Header.Add("Link", `<https://api.github.com/next>; rel=next`)
				case "extended relation":
					response.Header.Add("Link", `<https://api.github.com/next>; rel=next; rel*=utf-8''prev`)
				}
				return response, nil
			})
			if _, err := c.pruneList(runnerRepo); err == nil {
				t.Fatal("accepted incomplete inventory")
			}
		})
	}
}
