package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchiveReadRequiresCompleteTargetAndState(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, tt := range []struct {
		name, body string
		status     int
		ok         bool
	}{
		{"active", `{"id":42,"full_name":"owner/current","archived":false}`, 200, true},
		{"archived", `{"id":42,"full_name":"owner/current","archived":true}`, 200, true},
		{"missing ID", `{"full_name":"owner/current","archived":false}`, 200, false},
		{"fractional ID", `{"id":42.5,"full_name":"owner/current","archived":false}`, 200, false},
		{"zero ID", `{"id":0,"full_name":"owner/current","archived":false}`, 200, false},
		{"missing name", `{"id":42,"archived":false}`, 200, false},
		{"wrong name", `{"id":42,"full_name":"other/current","archived":false}`, 200, false},
		{"missing state", `{"id":42,"full_name":"owner/current"}`, 200, false},
		{"null state", `{"id":42,"full_name":"owner/current","archived":null}`, 200, false},
		{"string state", `{"id":42,"full_name":"owner/current","archived":"true"}`, 200, false},
		{"denied", `{"message":"unauthorized"}`, 403, false},
		{"not found", `{"message":"not found"}`, 404, false},
		{"redirect", `{"message":"moved"}`, 301, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			c := writeTestClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/repos/owner/current" {
					t.Fatal(r.Method, r.URL)
				}
				return archiveHTTP(r, tt.status, tt.body), nil
			})
			got, err := c.ReadArchive(repo)
			if (err == nil) != tt.ok || calls != 1 {
				t.Fatalf("state=%+v err=%v calls=%d", got, err, calls)
			}
			if tt.ok && (got.ID != 42 || got.FullName != "owner/current" || got.Archived != (tt.name == "archived")) {
				t.Fatal(got)
			}
		})
	}
}

func archiveHTTP(req *http.Request, status int, body string) *http.Response {
	return &http.Response{Request: req, StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestArchiveUpdateIsNarrowAndVerified(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	for _, target := range []bool{false, true} {
		t.Run(fmt.Sprint(target), func(t *testing.T) {
			reads, writes := 0, 0
			state := settings.ArchiveState{ID: 42, FullName: "owner/current", Archived: !target}
			c := writeTestClient(t, func(r *http.Request) (*http.Response, error) {
				switch r.Method {
				case "PATCH":
					writes++
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload) != 1 || payload["archived"] != target {
						t.Fatalf("payload=%v err=%v", payload, err)
					}
					return archiveHTTP(r, 200, fmt.Sprintf(`{"id":42,"full_name":"owner/current","archived":%t}`, target)), nil
				case "GET":
					reads++
					return archiveHTTP(r, 200, fmt.Sprintf(`{"id":42,"full_name":"owner/current","archived":%t}`, target)), nil
				default:
					t.Fatal("unexpected method", r.Method)
				}
				return nil, nil
			})
			if err := c.UpdateArchive(repo, state, target); err != nil || reads != 1 || writes != 1 {
				t.Fatal(err, reads, writes)
			}
		})
	}
}

func TestArchiveUpdateRejectsWrongResponseAndUnconfirmedReadback(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	base := settings.ArchiveState{ID: 42, FullName: "owner/current", Archived: false}
	for _, tt := range []struct {
		name, patch, read string
		status            int
		wantReads         int
	}{
		{"bad response ID", `{"id":41,"full_name":"owner/current","archived":true}`, ``, 200, 0},
		{"bad response target", `{"id":42,"full_name":"owner/other","archived":true}`, ``, 200, 0},
		{"ignored update", `{"id":42,"full_name":"owner/current","archived":false}`, ``, 200, 0},
		{"bad readback", `{"id":42,"full_name":"owner/current","archived":true}`, `{"id":42,"full_name":"owner/current","archived":false}`, 200, 1},
		{"renamed on readback", `{"id":42,"full_name":"owner/current","archived":true}`, `{"id":42,"full_name":"owner/other","archived":true}`, 200, 1},
		{"write denied", `{"message":"denied"}`, ``, 403, 0},
		{"write redirect", `{"message":"moved"}`, ``, 307, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			reads, writes := 0, 0
			c := writeTestClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method == "PATCH" {
					writes++
					return archiveHTTP(r, tt.status, tt.patch), nil
				}
				if r.Method == "GET" {
					reads++
					return archiveHTTP(r, 200, tt.read), nil
				}
				t.Fatal(r.Method)
				return nil, nil
			})
			if err := c.UpdateArchive(repo, base, true); err == nil || writes != 1 || reads != tt.wantReads {
				t.Fatal(tt.name, err, writes, reads)
			}
		})
	}
}

// A PATCH transport error cannot establish that the server did not mutate state.
func TestArchivePatchTransportFailureIsUncertain(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	base := settings.ArchiveState{ID: 42, FullName: "owner/current", Archived: false}
	calls := 0
	client := writeTestClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != "PATCH" {
			t.Fatalf("unexpected readback after transport failure: %s", req.Method)
		}
		return nil, fmt.Errorf("fixture lost response")
	})
	err := client.UpdateArchive(repo, base, true)
	if err == nil || calls != 1 || !strings.Contains(err.Error(), "write may have applied") || !strings.Contains(err.Error(), "fixture lost response") {
		t.Fatal(err, calls)
	}
}
