package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

var generalTestRepo = repository.Repository{Owner: "owner", Name: "current"}

const generalTestMetadata = `{"full_name":"owner/current","visibility":"private","owner":{"type":"Organization"}}`

func TestActionsCacheStrictReads(t *testing.T) {
	for _, kind := range []string{"retention", "storage"} {
		key := "max_cache_retention_days"
		if kind == "storage" {
			key = "max_cache_size_gb"
		}
		for _, body := range []string{`{}`, `null`, `[]`, fmt.Sprintf(`{"%s":null}`, key), fmt.Sprintf(`{"%s":"7"}`, key), fmt.Sprintf(`{"%s":1.5}`, key), fmt.Sprintf(`{"%s":0}`, key), fmt.Sprintf(`{"%s":10001}`, key)} {
			rest := actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					t.Fatal("read mutated", r.Method)
				}
				if r.URL.Path == "/repos/owner/current" {
					return actionsResponse(r, 200, generalTestMetadata), nil
				}
				if r.URL.Path != "/repos/owner/current/actions/cache/"+kind+"-limit" {
					t.Fatal(r.URL)
				}
				return actionsResponse(r, 200, body), nil
			}).rest
			var err error
			if kind == "retention" {
				_, err = (&CacheRetentionClient{rest: rest}).Read(generalTestRepo)
			} else {
				_, err = (&CacheStorageClient{rest: rest}).Read(generalTestRepo)
			}
			if err == nil {
				t.Fatal(kind, body, "accepted")
			}
		}
	}
}
func TestActionsGeneralFreshNarrowWriteAndVerification(t *testing.T) {
	for _, kind := range []string{"retention", "storage", "access"} {
		for _, failure := range []string{"", "read", "write", "ignored", "verify", "metadata", "metadata after", "noop"} {
			t.Run(kind+"/"+failure, func(t *testing.T) {
				key, path := "max_cache_retention_days", "/repos/owner/current/actions/cache/retention-limit"
				var before, desired any = float64(7), float64(30)
				if kind == "storage" {
					key, path = "max_cache_size_gb", "/repos/owner/current/actions/cache/storage-limit"
				}
				if kind == "access" {
					key, path, before, desired = "access_level", "/repos/owner/current/actions/permissions/access", "none", "organization"
				}
				if failure == "noop" {
					before = desired
				}
				writes, reads := 0, 0
				rest := actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == "/repos/owner/current" {
						if r.Method != "GET" {
							t.Fatal(r.Method)
						}
						if failure == "metadata" || failure == "metadata after" && writes > 0 {
							return actionsResponse(r, 200, `{"full_name":"other/repo","visibility":"private"}`), nil
						}
						return actionsResponse(r, 200, generalTestMetadata), nil
					}
					if r.URL.Path != path {
						t.Fatal("unrelated", r.URL)
					}
					if r.Method == "GET" {
						reads++
						if failure == "read" || writes > 0 && failure == "verify" {
							return actionsResponse(r, 403, `{}`), nil
						}
						body, _ := json.Marshal(map[string]any{key: before})
						return actionsResponse(r, 200, string(body)), nil
					}
					if r.Method != "PUT" {
						t.Fatal(r.Method)
					}
					if reads != 1 {
						t.Fatal("write without fresh read", reads)
					}
					writes++
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(body, map[string]any{key: desired}) {
						t.Fatal("wrong payload", body)
					}
					if failure == "write" {
						return actionsResponse(r, 422, `{}`), nil
					}
					if failure != "ignored" {
						before = desired
					}
					return actionsResponse(r, 204, ""), nil
				}).rest
				update := generalUpdate(kind, rest)
				wrote, err := update()
				metadataFailure := kind != "storage" && (failure == "metadata" || failure == "metadata after")
				good := failure == "" || failure == "noop" || kind == "storage" && (failure == "metadata" || failure == "metadata after")
				if (err == nil) != good || wrote != (good && failure != "noop") {
					t.Fatal(wrote, err)
				}
				var pre interface{ ReadFailedBeforeWrite() }
				if errors.As(err, &pre) != (failure == "read" || metadataFailure && failure == "metadata") {
					t.Fatal("wrong effect certainty", err)
				}
				if (failure == "read" || metadataFailure && failure == "metadata" || failure == "noop") && writes != 0 {
					t.Fatal("unexpected mutation", writes)
				}
				if good && failure != "noop" && reads != 2 {
					t.Fatal("missing independent readback", reads)
				}
			})
		}
	}
}
func generalUpdate(kind string, rest *api.RESTClient) func() (bool, error) {
	if kind == "retention" {
		return func() (bool, error) { return (&CacheRetentionClient{rest: rest}).Update(generalTestRepo, 30) }
	}
	if kind == "storage" {
		return func() (bool, error) { return (&CacheStorageClient{rest: rest}).Update(generalTestRepo, 30) }
	}
	return func() (bool, error) {
		return (&ActionsAccessClient{rest: rest}).Update(generalTestRepo, "organization")
	}
}
func TestActionsGeneralRejectsInvalidDesiredWithoutIO(t *testing.T) {
	rest := actionsClient(t, func(r *http.Request) (*http.Response, error) {
		t.Fatal("invalid desired accessed API", r.URL)
		return nil, nil
	}).rest
	for _, v := range []int{0, 366} {
		if _, err := (&CacheRetentionClient{rest: rest}).Update(generalTestRepo, v); err == nil {
			t.Fatal(v)
		}
	}
	for _, v := range []int{0, 10001} {
		if _, err := (&CacheStorageClient{rest: rest}).Update(generalTestRepo, v); err == nil {
			t.Fatal(v)
		}
	}
	if _, err := (&ActionsAccessClient{rest: rest}).Update(generalTestRepo, "enterprise"); err == nil {
		t.Fatal("enterprise accepted")
	}
}

func TestCacheRetentionPostWriteVisibilityChange(t *testing.T) {
	writes := 0
	client := &CacheRetentionClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/repos/owner/current" {
			visibility := "private"
			if writes > 0 {
				visibility = "public"
			}
			return actionsResponse(r, 200, fmt.Sprintf(`{"full_name":"owner/current","visibility":"%s"}`, visibility)), nil
		}
		if r.URL.Path != "/repos/owner/current/actions/cache/retention-limit" {
			t.Fatal(r.URL)
		}
		if r.Method == "PUT" {
			writes++
			return actionsResponse(r, 204, ""), nil
		}
		days := 7
		if writes > 0 {
			days = 365
		}
		return actionsResponse(r, 200, fmt.Sprintf(`{"max_cache_retention_days":%d}`, days)), nil
	}).rest}
	wrote, err := client.Update(generalTestRepo, 365)
	var pre interface{ ReadFailedBeforeWrite() }
	if wrote || err == nil || writes != 1 || errors.As(err, &pre) {
		t.Fatal(wrote, writes, err)
	}
}

func TestActionsCacheRejectsAmbiguousOrOversizedObjects(t *testing.T) {
	for _, tc := range []struct{ kind, metadata, body string }{
		{"retention", generalTestMetadata, `{"max_cache_retention_days":5,"max_cache_retention_days":7}`},
		{"storage", generalTestMetadata, `{"max_cache_size_gb":5,"max_cache_size_gb":7}`},
		{"retention", generalTestMetadata, `{"max_cache_retention_days":7,"MAX_CACHE_RETENTION_DAYS":5}`},
		{"retention", generalTestMetadata, `{"MAX_CACHE_RETENTION_DAYS":5,"max_cache_retention_days":7}`},
		{"storage", generalTestMetadata, `{"max_cache_size_gb":7,"MAX_CACHE_SIZE_GB":5}`},
		{"storage", generalTestMetadata, `{"MAX_CACHE_SIZE_GB":5,"max_cache_size_gb":7}`},
		{"retention", `{"full_name":"owner/current","FULL_NAME":"other/repo","visibility":"public"}`, `{"max_cache_retention_days":7}`},
		{"retention", `{"FULL_NAME":"other/repo","full_name":"owner/current","visibility":"public"}`, `{"max_cache_retention_days":7}`},
		{"retention", `{"full_name":"owner/current","visibility":"public","VISIBILITY":"private"}`, `{"max_cache_retention_days":7}`},
		{"retention", `{"full_name":"owner/current","VISIBILITY":"private","visibility":"public"}`, `{"max_cache_retention_days":7}`},
		{"storage", generalTestMetadata, `{"max_cache_size_gb":7,"extra":"` + string([]byte{0xff}) + `"}`},
		{"retention", `{"full_name":"wrong/repo","full_name":"owner/current","visibility":"public"}`, `{"max_cache_retention_days":7}`},
		{"retention", `{"full_name":"owner/current","visibility":"private","visibility":"public"}`, `{"max_cache_retention_days":7}`},
		{"storage", generalTestMetadata, `{"max_cache_size_gb":7} {}`},
		{"storage", generalTestMetadata, `{"max_cache_size_gb":7,"padding":"` + strings.Repeat("x", 1<<20) + `"}`},
	} {
		t.Run(tc.kind+"/"+fmt.Sprint(len(tc.body))+"/"+tc.metadata, func(t *testing.T) {
			rest := actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatal("unexpected write")
				}
				if r.URL.Path == "/repos/owner/current" {
					return actionsResponse(r, 200, tc.metadata), nil
				}
				return actionsResponse(r, 200, tc.body), nil
			}).rest
			var err error
			if tc.kind == "retention" {
				_, err = (&CacheRetentionClient{rest: rest}).Read(generalTestRepo)
			} else {
				_, err = (&CacheStorageClient{rest: rest}).Read(generalTestRepo)
			}
			if err == nil {
				t.Fatal("ambiguous or oversized response accepted")
			}
		})
	}
}

func TestActionsAccessRejectsDuplicateSharedMetadata(t *testing.T) {
	rest := actionsClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.Path != "/repos/owner/current" {
			t.Fatal("unexpected access request", r.Method, r.URL.Path)
		}
		return actionsResponse(r, 200, `{"full_name":"owner/current","visibility":"public","visibility":"private","owner":{"type":"User"}}`), nil
	}).rest
	if _, err := (&ActionsAccessClient{rest: rest}).Read(generalTestRepo); err == nil {
		t.Fatal("ambiguous shared metadata accepted")
	}
}

func TestActionsAccessRejectsAmbiguousOwnerEvidence(t *testing.T) {
	for _, metadata := range []string{
		`{"full_name":"owner/current","visibility":"private","owner":{"type":"Organization"},"OWNER":{"type":"User"}}`,
		`{"full_name":"owner/current","visibility":"private","OWNER":{"type":"User"},"owner":{"type":"Organization"}}`,
		`{"full_name":"owner/current","visibility":"private","owner":{"type":"Organization","TYPE":"User"}}`,
		`{"full_name":"owner/current","visibility":"private","owner":{"TYPE":"User","type":"Organization"}}`,
	} {
		t.Run(metadata, func(t *testing.T) {
			reads, writes := 0, 0
			rest := actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					writes++
					t.Fatal("unexpected write")
				}
				reads++
				if r.URL.Path != "/repos/owner/current" {
					t.Fatal("access read before identity validation")
				}
				return actionsResponse(r, 200, metadata), nil
			}).rest
			wrote, err := (&ActionsAccessClient{rest: rest}).Update(generalTestRepo, "organization")
			var pre interface{ ReadFailedBeforeWrite() }
			if err == nil || wrote || !errors.As(err, &pre) || writes != 0 || reads != 1 {
				t.Fatal(wrote, err, reads, writes)
			}
		})
	}
}
