package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedAutolinkMatching(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, mode := range []string{"match", "unselected drift", "prefix", "url", "format", "missing", "incomplete", "denied first", "denied fresh", "reader nil", "repo id", "repo name", "owner id", "archive", "final repo", "timestamp", "create", "prune", "mixed", "mixed differs"} {
		t.Run(mode, func(t *testing.T) {
			yes := true
			values := []settings.Autolink{appAutolink(7, "OLD-")}
			if mode == "create" {
				values[0].ID = nil
			}
			initial := settings.CloneAutolinks(values)
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Autolinks: &values}
			if mode == "mixed" || mode == "mixed differs" {
				d.Topics.Names = &[]string{"one"}
			}
			reads, archiveReads, topicReads, writes, bindings := 0, 0, 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archiveReads++
					id := int64(1)
					if mode == "final repo" && archiveReads == 2 {
						id = 9
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				ReadTopics: func(repository.Repository) ([]string, error) {
					topicReads++
					if mode == "mixed differs" {
						return []string{"two"}, nil
					}
					return []string{"one"}, nil
				},
				ReadAutolinks: func(repository.Repository) (settings.AutolinkObservation, error) {
					reads++
					if mode == "denied first" || mode == "denied fresh" && reads == 2 {
						return settings.AutolinkObservation{}, errors.New("denied")
					}
					o := appAutolinks()
					o.Target.Archived = true
					if reads == 2 {
						switch mode {
						case "unselected drift":
							o.Entries = append(o.Entries, settings.AutolinkState{Value: appAutolink(8, "OTHER-")})
						case "prefix":
							o.Entries[0].Value.ReferencePrefix = "DIFF-"
						case "url":
							o.Entries[0].Value.TargetURL = "https://other/<num>"
						case "format":
							o.Entries[0].Value.ReferenceFormat = "alphanumeric"
						case "missing":
							o.Entries = []settings.AutolinkState{}
						case "incomplete":
							o.Entries = nil
						case "repo id":
							o.Target.ID = 9
						case "repo name":
							o.Target.FullName = "owner/other"
						case "owner id":
							o.Target.OwnerID = 9
						case "archive":
							o.Target.Archived = false
						case "timestamp":
							o.Entries[0].UpdatedAtPresent = true
						}
					}
					return o, nil
				},
				CreateAutolink: func(repository.Repository, settings.AutolinkObservation, settings.Autolink) (settings.AutolinkWriteResult, error) {
					writes++
					return settings.AutolinkWriteResult{}, nil
				},
				DeleteAutolink: func(repository.Repository, settings.AutolinkObservation, int64) (bool, error) {
					writes++
					return false, nil
				},
				CreatedID: func(string, string, int64) error { bindings++; return nil },
				Archive:   func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
			}
			if mode == "reader nil" {
				w.ReadAutolinks = nil
			}
			var prune settings.PruneScopes
			if mode == "prune" {
				var err error
				prune, err = settings.ParsePruneScopes("autolinks")
				if err != nil {
					t.Fatal(err)
				}
			}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w, prune)
			want := mode == "match" || mode == "unselected drift" || mode == "mixed"
			if (err == nil) != want || writes != 0 || bindings != 0 || len(result.CreatedIDs) != 0 {
				t.Fatalf("err=%v writes=%d bindings=%d ids=%v", err, writes, bindings, result.CreatedIDs)
			}
			if !reflect.DeepEqual(values, initial) {
				t.Fatal("modified declaration")
			}
			wantReads, wantArchive := 2, 1
			switch mode {
			case "match", "unselected drift", "mixed", "final repo":
				wantArchive = 2
			case "denied first":
				wantReads = 1
			case "create", "reader nil", "mixed differs":
				wantReads = 0
			case "prune":
				wantReads = 1 // Global prune preflight precedes the archive guard.
				wantArchive = 0
			}
			if reads != wantReads || archiveReads != wantArchive {
				t.Fatalf("reads=%d archive=%d", reads, archiveReads)
			}
			if (mode == "mixed" || mode == "mixed differs") && topicReads != 1 {
				t.Fatalf("topics reads=%d", topicReads)
			}
			if mode == "create" && !strings.Contains(err.Error(), "existing IDs required") || mode == "reader nil" && !strings.Contains(err.Error(), "reader unavailable") || mode == "mixed differs" && !strings.Contains(err.Error(), "topics differ") || mode == "prune" && !strings.Contains(err.Error(), "autolink prune requires explicit unarchive") {
				t.Fatalf("unexpected failure: %v", err)
			}
		})
	}
}
