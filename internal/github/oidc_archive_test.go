package github

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedOIDCReadsBindRepositoryAndOwner(t *testing.T) {
	for _, mode := range []string{"matching", "repo id first", "repo id fresh", "owner fresh", "unarchived fresh", "repo missing", "subject denied", "subject malformed", "duplicate repository id", "duplicate owner id", "duplicate mode", "duplicate keys", "duplicate immutable"} {
		t.Run(mode, func(t *testing.T) {
			repo := repository.Repository{Owner: "owner", Name: "current"}
			archive := settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}
			repoReads, subjectReads, writes := 0, 0, 0
			c := &OIDCSubjectClient{actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" {
					writes++
					return aiRepositoryResponse(400, `{}`), nil
				}
				if r.URL.Path == "/repos/owner/current" {
					repoReads++
					target := strings.Replace(oidcTarget, `"archived":false`, `"archived":true`, 1)
					if mode == "repo id first" || mode == "repo id fresh" && repoReads == 2 {
						target = strings.Replace(target, `"id":1`, `"id":3`, 1)
					}
					if mode == "owner fresh" && repoReads == 2 {
						target = strings.Replace(target, `"id":2`, `"id":3`, 1)
					}
					if mode == "unarchived fresh" && repoReads == 2 {
						target = oidcTarget
					}
					if mode == "duplicate repository id" && repoReads == 2 {
						target = strings.Replace(target, `"id":1`, `"id":9,"id":1`, 1)
					}
					if mode == "duplicate owner id" && repoReads == 2 {
						target = strings.Replace(target, `"id":2`, `"id":9,"id":2`, 1)
					}
					if mode == "repo missing" {
						target = `{}`
					}
					return aiRepositoryResponse(200, target), nil
				}
				if r.URL.Path != "/repos/owner/current/actions/oidc/customization/sub" {
					t.Fatal(r.URL)
				}
				subjectReads++
				if mode == "subject denied" && subjectReads == 2 {
					return aiRepositoryResponse(403, `{"message":"PRIVATE_CANARY"}`), nil
				}
				body := `{"use_default":false,"include_claim_keys":["repo","context"],"use_immutable_subject":false}`
				if mode == "duplicate mode" && subjectReads == 2 {
					body = `{"use_default":true,"use_default":false,"include_claim_keys":["repo","context"],"use_immutable_subject":false}`
				}
				if mode == "duplicate keys" && subjectReads == 2 {
					body = `{"use_default":false,"include_claim_keys":["other"],"include_claim_keys":["repo","context"],"use_immutable_subject":false}`
				}
				if mode == "duplicate immutable" && subjectReads == 2 {
					body = `{"use_default":false,"include_claim_keys":["repo","context"],"use_immutable_subject":true,"use_immutable_subject":false}`
				}
				if mode == "subject malformed" && subjectReads == 2 {
					body = `{"use_default":false}`
				}
				return aiRepositoryResponse(200, body), nil
			}).rest}
			observations, err := c.ReadArchived(repo, archive)
			if (err == nil) != (mode == "matching") || writes != 0 {
				t.Fatalf("err=%v writes=%d", err, writes)
			}
			if err == nil {
				no := false
				keys := []string{"repo", "context"}
				d := settings.OIDCSubjectTemplate{UseDefaultTemplate: &no, SubjectClaimTemplate: &keys, UseImmutableSubjectClaim: &no}
				for _, o := range observations {
					managed, changes, e := settings.CompareOIDCSubject(o, &d)
					if e != nil || managed != 1 || len(changes) != 0 {
						t.Fatalf("invalid evidence: %v", e)
					}
				}
				if repoReads != 2 || subjectReads != 2 {
					t.Fatalf("reads: %d %d", repoReads, subjectReads)
				}
			}
		})
	}
}

func TestArchivedOIDCRejectsIdentityFieldAliases(t *testing.T) {
	for _, change := range []struct {
		name, from, to string
	}{
		{"repository id", `"id":1`, `"ID":9,"id":1`},
		{"archived", `"archived":true`, `"Archived":false,"archived":true`},
		{"full name", `"full_name":"owner/current"`, `"Full_Name":"wrong/repo","full_name":"owner/current"`},
		{"owner id", `"id":2`, `"ID":9,"id":2`},
		{"owner type", `"type":"User"`, `"Type":"Organization","type":"User"`},
	} {
		for _, targetRead := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/read_%d", change.name, targetRead), func(t *testing.T) {
				repoReads, subjectReads := 0, 0
				c := &OIDCSubjectClient{actionsClient(t, func(r *http.Request) (*http.Response, error) {
					if r.Method != http.MethodGet {
						t.Fatalf("unexpected method: %s", r.Method)
					}
					if r.URL.Path == "/repos/owner/current" {
						repoReads++
						target := strings.Replace(oidcTarget, `"archived":false`, `"archived":true`, 1)
						if repoReads == targetRead {
							target = strings.Replace(target, change.from, change.to, 1)
						}
						return aiRepositoryResponse(200, target), nil
					}
					if r.URL.Path != "/repos/owner/current/actions/oidc/customization/sub" {
						t.Fatalf("unexpected route: %s", r.URL.Path)
					}
					subjectReads++
					return aiRepositoryResponse(200, `{"use_default":true,"use_immutable_subject":false}`), nil
				}).rest}
				_, err := c.ReadArchived(repository.Repository{Owner: "owner", Name: "current"}, settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true})
				if err == nil || repoReads != targetRead || subjectReads != targetRead-1 {
					t.Fatalf("err=%v repository reads=%d subject reads=%d", err, repoReads, subjectReads)
				}
			})
		}
	}
}
