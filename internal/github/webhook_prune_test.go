package github

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func webhookPruneRow(id int64) map[string]any {
	row := webhookRow(id, webhookCfg())
	row["updated_at"] = "2026-11-14T01:00:00Z"
	return row
}
func TestWebhookPruneCompleteAcquisitionAndDeletion(t *testing.T) {
	for _, mode := range []string{"success", "secret-mask", "secret-absent", "delivery-change", "event-order", "retained-timestamp", "candidate-timestamp", "retained-config", "unknown-parent", "unknown-config", "missing-timestamp", "malformed-timestamp", "addition", "removal", "duplicate", "short-next", "extended-next", "parent-config", "candidate-final-drift", "target-drift", "HTTP-error", "404", "wrong-status", "nonempty204", "no-removal", "absence-denied", "after-retained-drift", "after-addition"} {
		t.Run(mode, func(t *testing.T) {
			rows := map[int64]map[string]any{7: webhookPruneRow(7), 8: webhookPruneRow(8)}
			prepared, deletes, details8 := false, 0, 0
			c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodDelete {
					deletes++
					if req.URL.Path != "/repos/owner/current/hooks/8" || req.Body != nil {
						t.Fatal("incorrect deletion route or body", req.URL)
					}
					switch mode {
					case "HTTP-error":
						return collaboratorsHTTPResponse(req, 500, `{"message":"PRIVATE_DELETE_ERROR"}`), nil
					case "404":
						return collaboratorsHTTPResponse(req, 404, `{"message":"PRIVATE_DELETE_ERROR"}`), nil
					case "wrong-status":
						return collaboratorsHTTPResponse(req, 200, ``), nil
					case "nonempty204":
						return collaboratorsHTTPResponse(req, 204, `PRIVATE_BODY`), nil
					}
					if mode != "no-removal" {
						delete(rows, 8)
					}
					if mode == "after-retained-drift" {
						rows[7]["updated_at"] = "2026-11-14T03:00:00Z"
					}
					if mode == "after-addition" {
						rows[9] = webhookPruneRow(9)
					}
					return collaboratorsHTTPResponse(req, 204, ""), nil
				}
				if req.Method != http.MethodGet {
					t.Fatal("unexpected mutation", req.Method)
				}
				if req.URL.Path == "/repos/owner/current" {
					body := webhookTargetJSON
					if prepared && mode == "target-drift" {
						body = strings.Replace(body, `"id":1`, `"id":99`, 1)
					}
					return collaboratorsHTTPResponse(req, 200, body), nil
				}
				if req.URL.Path == "/repos/owner/current/hooks" {
					if req.URL.RawQuery != "per_page=100&page=1" {
						t.Fatal("unexpected page route", req.URL)
					}
					if deletes > 0 && mode == "absence-denied" {
						return collaboratorsHTTPResponse(req, 403, `{"message":"PRIVATE_ABSENCE_ERROR"}`), nil
					}
					list := []any{}
					for _, id := range []int64{7, 8, 9} {
						if row, ok := rows[id]; ok {
							list = append(list, row)
						}
					}
					if prepared && mode == "duplicate" {
						list = append(list, rows[7])
					}
					response := collaboratorsHTTPResponse(req, 200, webhookBody(list))
					if prepared && mode == "short-next" {
						response.Header.Set("Link", `<https://foreign.example/PRIVATE>; rel=next`)
					}
					if prepared && mode == "extended-next" {
						response.Header.Set("Link", `<https://foreign.example/PRIVATE>; rel=next; rel*=utf-8''prev`)
					}
					return response, nil
				}
				path := strings.TrimPrefix(req.URL.Path, "/repos/owner/current/hooks/")
				parts := strings.Split(path, "/")
				id, err := strconv.ParseInt(parts[0], 10, 64)
				if err != nil || rows[id] == nil || len(parts) > 2 || len(parts) == 2 && parts[1] != "config" {
					t.Fatal("unexpected fixed route", req.URL)
				}
				if len(parts) == 2 {
					cfg := rows[id]["config"]
					if prepared && mode == "parent-config" && id == 7 {
						cfg = map[string]any{"url": "https://other.test", "content_type": "json", "insecure_ssl": 0}
					}
					return collaboratorsHTTPResponse(req, 200, webhookBody(cfg)), nil
				}
				if id == 8 {
					details8++
					if prepared && mode == "candidate-final-drift" && details8 == 3 {
						rows[8]["updated_at"] = "2026-11-14T02:00:00Z"
					}
				}
				return collaboratorsHTTPResponse(req, 200, webhookBody(rows[id])), nil
			})
			observed, err := c.ReadWebhookPrune(webhookRepo)
			if err != nil {
				t.Fatal(err)
			}
			mime := "application/json"
			expected, err := settings.PrepareWebhookPrune(observed, []settings.Webhook{{ID: 7, ContentType: &mime}})
			if err != nil {
				t.Fatal(err)
			}
			prepared = true
			switch mode {
			case "secret-mask":
				rows[7]["config"].(map[string]any)["secret"] = "********"
			case "secret-absent":
				delete(rows[8]["config"].(map[string]any), "secret")
			case "delivery-change":
				rows[7]["last_response"] = map[string]any{"message": "PRIVATE_NEW_RESPONSE"}
				rows[8]["test_url"] = "https://never-follow.test/PRIVATE"
			case "event-order":
				rows[7]["events"] = []string{"push", "future_event"}
			case "retained-timestamp":
				rows[7]["updated_at"] = "2026-11-14T02:00:00Z"
			case "candidate-timestamp":
				rows[8]["updated_at"] = "2026-11-14T02:00:00Z"
			case "retained-config":
				rows[7]["config"].(map[string]any)["url"] = "https://changed.test/PRIVATE"
			case "unknown-parent":
				rows[7]["PRIVATE_NEW_CONFIGURATION"] = true
			case "unknown-config":
				rows[8]["config"].(map[string]any)["PRIVATE_NEW_CONFIGURATION"] = true
			case "missing-timestamp":
				delete(rows[8], "updated_at")
			case "malformed-timestamp":
				rows[8]["updated_at"] = "PRIVATE_INVALID_TIMESTAMP"
			case "addition":
				rows[9] = webhookPruneRow(9)
			case "removal":
				delete(rows, 8)
			}
			attempted, err := c.DeleteWebhook(webhookRepo, expected, 8)
			success := mode == "success" || mode == "secret-mask" || mode == "secret-absent" || mode == "delivery-change" || mode == "event-order"
			post := mode == "HTTP-error" || mode == "404" || mode == "wrong-status" || mode == "nonempty204" || mode == "no-removal" || mode == "absence-denied" || mode == "after-retained-drift" || mode == "after-addition"
			if (err == nil) != success || attempted != (success || post) || deletes != boolCount(success || post) {
				t.Fatal("dishonest progress", attempted, deletes, err)
			}
			if err != nil {
				var pre *securityPreWriteFailure
				if errors.As(err, &pre) == post || strings.Contains(err.Error(), "PRIVATE") {
					t.Fatal("failure classification/privacy", err)
				}
			}
		})
	}
}
func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}
func TestWebhookPruneUnknownParentDoesNotNarrowOrdinaryReads(t *testing.T) {
	c := topicsClient(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/repos/owner/current" {
			return collaboratorsHTTPResponse(req, 200, webhookTargetJSON), nil
		}
		if strings.HasSuffix(req.URL.Path, "/config") {
			return collaboratorsHTTPResponse(req, 200, webhookBody(webhookCfg())), nil
		}
		row := webhookPruneRow(7)
		row["PRIVATE_UNKNOWN_PARENT"] = true
		if req.URL.Path == "/repos/owner/current/hooks" {
			return collaboratorsHTTPResponse(req, 200, webhookBody([]any{row})), nil
		}
		if req.URL.Path != "/repos/owner/current/hooks/7" {
			t.Fatal(req.URL)
		}
		return collaboratorsHTTPResponse(req, 200, webhookBody(row)), nil
	})
	if _, err := c.ReadWebhooks(webhookRepo, []int64{7}); err != nil {
		t.Fatal("ordinary selected config behavior changed", err)
	}
	if _, err := c.ReadWebhookPrune(webhookRepo); err == nil || strings.Contains(fmt.Sprint(err), "PRIVATE") {
		t.Fatal("unknown destructive evidence accepted or leaked", err)
	}
}

func TestWebhookPruneInventoryPagesAndEmptyTargetBinding(t *testing.T) {
	for _, mode := range []string{"complete-pages", "repeated-page", "empty-replacement", "changed-during-acquisition"} {
		t.Run(mode, func(t *testing.T) {
			pages, details, targetID := 0, 0, int64(1)
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path == "/repos/owner/current" {
					return collaboratorsHTTPResponse(req, 200, strings.Replace(webhookTargetJSON, `"id":1`, fmt.Sprintf(`"id":%d`, targetID), 1)), nil
				}
				if req.URL.Path == "/repos/owner/current/hooks" {
					pages++
					page := (pages-1)%2 + 1
					if mode == "empty-replacement" {
						page = 1
					}
					if req.URL.RawQuery != fmt.Sprintf("per_page=100&page=%d", page) {
						t.Fatal("pagination followed supplied URL", req.URL)
					}
					rows := []any{}
					if mode == "empty-replacement" {
						targetID = 99
					} else if page == 1 || mode == "repeated-page" {
						for id := int64(1); id <= 100; id++ {
							row := webhookPruneRow(id)
							if mode == "changed-during-acquisition" && pages > 2 && id == 1 {
								row["updated_at"] = "2026-11-14T02:00:00Z"
							}
							rows = append(rows, row)
						}
					}
					response := collaboratorsHTTPResponse(req, 200, webhookBody(rows))
					if page == 1 && mode != "empty-replacement" {
						response.Header.Set("Link", `<https://foreign.example/PRIVATE_PAGE>; rel=next`)
					}
					return response, nil
				}
				details++
				parts := strings.Split(strings.TrimPrefix(req.URL.Path, "/repos/owner/current/hooks/"), "/")
				id, e := strconv.ParseInt(parts[0], 10, 64)
				if e != nil || id < 1 || id > 100 {
					t.Fatal("unexpected detail route", req.URL)
				}
				if len(parts) == 2 && parts[1] == "config" {
					return collaboratorsHTTPResponse(req, 200, webhookBody(webhookCfg())), nil
				}
				if len(parts) != 1 {
					t.Fatal("unexpected child route", req.URL)
				}
				return collaboratorsHTTPResponse(req, 200, webhookBody(webhookPruneRow(id))), nil
			})
			observed, err := client.ReadWebhookPrune(webhookRepo)
			if mode == "complete-pages" {
				if err != nil || pages != 4 || details != 200 || len(observed.Inventory().Values) != 100 {
					t.Fatal(pages, details, err)
				}
			} else if err == nil || (mode != "changed-during-acquisition" && details != 0) {
				t.Fatal("incomplete or unbound inventory accepted", details, err)
			}
		})
	}
}
