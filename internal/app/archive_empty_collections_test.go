package app

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// The selected empty collections are unmanaged only without pruning. Each
// nonempty entry is valid and must stay outside archived matching until its
// resource-specific reader and comparator are explicitly supported.
func TestArchivedMatchingUsesOrdinaryEmptyCollectionSelection(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	yes, desc, content, color := true, "same", "application/json", "112233"
	key, err := settings.ParseDeployPublicKey(appDeployOne)
	if err != nil {
		t.Fatal(err)
	}
	emptyKeys, err := settings.NewDeployKeyDeclarations(nil)
	if err != nil {
		t.Fatal(err)
	}
	fullKeys, err := settings.NewDeployKeyDeclarations([]settings.DeployKeyDeclaration{{Title: "key", Key: key}})
	if err != nil {
		t.Fatal(err)
	}
	emptyCollaborators, err := settings.NewCollaboratorDeclarations(nil)
	if err != nil {
		t.Fatal(err)
	}
	fullCollaborators, err := settings.NewCollaboratorDeclarations([]settings.CollaboratorDeclaration{{Username: "person"}})
	if err != nil {
		t.Fatal(err)
	}
	emptyVars, err := settings.NewVariableDeclarations(nil)
	if err != nil {
		t.Fatal(err)
	}
	fullVars, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "SAMPLE", Value: "value"}})
	if err != nil {
		t.Fatal(err)
	}
	emptyEnvs, err := settings.NewEnvironmentDeclarations(nil)
	if err != nil {
		t.Fatal(err)
	}
	fullEnvs, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "production"}})
	if err != nil {
		t.Fatal(err)
	}
	emptyPolicies := eventPolicyDeclaration(t)
	fullPolicies := eventPolicyDeclaration(t, eventAppPolicy(7, "policy"))
	emptyRules := appRulesets(t, `[]`)
	fullRules := appRulesets(t, `[{id: 7, name: basic, enforcement: disabled, conditions: {ref_name: {include: ['~ALL'], exclude: []}}, bypass_actors: [], rules: [{type: deletion}]}]`)
	emptyLabels, err := settings.NewLabelDeclarations(nil)
	if err != nil {
		t.Fatal(err)
	}
	labelID := int64(7)
	fullLabels, err := settings.NewLabelDeclarations([]settings.LabelDeclaration{{ID: &labelID, Name: "bug", Color: &color}})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name              string
		set               func(*settings.Declaration, bool)
		supportedNonempty bool
	}{
		{"webhooks", func(d *settings.Declaration, full bool) {
			v := []settings.Webhook{}
			if full {
				v = append(v, settings.Webhook{ID: 7, ContentType: &content})
			}
			d.Webhooks = &v
		}, false},
		{"runners", func(d *settings.Declaration, full bool) {
			v := []settings.RunnerLabels{}
			if full {
				v = append(v, appRunner(7, "gpu"))
			}
			d.Runners = &v
		}, false},
		{"deploy keys", func(d *settings.Declaration, full bool) {
			if full {
				d.DeployKeys = &fullKeys
			} else {
				d.DeployKeys = &emptyKeys
			}
		}, false},
		{"autolinks", func(d *settings.Declaration, full bool) {
			v := []settings.Autolink{}
			if full {
				v = append(v, appAutolink(7, "DOC-"))
			}
			d.Autolinks = &v
		}, false},
		{"classic protection", func(d *settings.Declaration, full bool) {
			v := []settings.ClassicRule{}
			if full {
				v = append(v, settings.ClassicRule{BranchNamePattern: "main"})
			}
			d.BranchProtectionRules = &v
		}, false},
		{"collaborators", func(d *settings.Declaration, full bool) {
			if full {
				d.Collaborators = &fullCollaborators
			} else {
				d.Collaborators = &emptyCollaborators
			}
		}, false},
		{"actions policies", func(d *settings.Declaration, full bool) {
			if full {
				d.ActionsPolicies = fullPolicies
			} else {
				d.ActionsPolicies = emptyPolicies
			}
		}, false},
		{"variables", func(d *settings.Declaration, full bool) {
			if full {
				d.Variables = &fullVars
			} else {
				d.Variables = &emptyVars
			}
		}, false},
		{"agents variables", func(d *settings.Declaration, full bool) {
			if full {
				d.AgentsVariables = &fullVars
			} else {
				d.AgentsVariables = &emptyVars
			}
		}, false},
		{"rulesets", func(d *settings.Declaration, full bool) {
			if full {
				d.Rulesets = fullRules
			} else {
				d.Rulesets = emptyRules
			}
		}, false},
		{"environments", func(d *settings.Declaration, full bool) {
			if full {
				d.Environments = &fullEnvs
			} else {
				d.Environments = &emptyEnvs
			}
		}, false},
		{"environment variable parent", func(d *settings.Declaration, full bool) {
			v := emptyVars
			if full {
				v = fullVars
			}
			d.EnvironmentVariables = []settings.EnvironmentVariableDeclaration{{Environment: "production", Variables: v}}
		}, false},
		{"labels", func(d *settings.Declaration, full bool) {
			if full {
				d.Labels = &fullLabels
			} else {
				d.Labels = &emptyLabels
			}
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("target-prune", func(t *testing.T) {
				scopes := map[string]string{
					"webhooks": "webhooks", "runners": "runners", "deploy keys": "deploy-keys",
					"autolinks": "autolinks", "classic protection": "branch-protection-rules",
					"collaborators": "collaborators", "actions policies": "actions-policies",
					"variables": "actions-variables", "agents variables": "agents-variables",
					"rulesets": "rulesets", "environments": "environments",
					"environment variable parent": "environment-variables", "labels": "labels",
				}
				prune, err := settings.ParsePruneScopes(scopes[tc.name])
				if err != nil {
					t.Fatal(err)
				}
				d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes, Description: &desc}}
				tc.set(&d, false)
				if !prune.Manages(d) || archivedMatchingScope(d, prune) {
					t.Fatal("targeted prune must remain managed and reject read-only matching")
				}
			})
			for _, mode := range []string{"omitted", "empty", "normal-empty", "nonempty", "prune", "denied"} {
				t.Run(mode, func(t *testing.T) {
					d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes, Description: &desc}}
					if mode == "normal-empty" {
						d.Repository.Archived = nil
					}
					if mode != "omitted" {
						tc.set(&d, mode == "nonempty")
					}
					before := d
					prune := settings.PruneScopes{}
					if mode == "prune" {
						var e error
						prune, e = settings.ParsePruneScopes("labels")
						if e != nil {
							t.Fatal(e)
						}
					}
					reads, writes, labelReads, repositoryCalls := 0, 0, 0, 0
					w := Writer{
						ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
							reads++
							return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
						},
						ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
							if mode == "denied" {
								return settings.RepositoryObservation{}, errors.New("denied")
							}
							return settings.NewRepositoryObservation(&settings.RepositoryState{Description: desc}, nil, nil), nil
						},
						ReadLabels: func(repository.Repository) (settings.LabelObservation, error) {
							labelReads++
							if mode == "nonempty" && tc.supportedNonempty {
								return settings.NewLabelObservation([]settings.LabelState{{ID: 7, Name: "bug", Color: color}})
							}
							return settings.NewLabelObservation(nil)
						},
						Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
							repositoryCalls++
							return false, nil
						},
						Archive:            func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
						CreatedID:          func(string, string, int64) error { writes++; return nil },
						BindCollaboratorID: func(string, int64) error { writes++; return nil },
						DeleteLabel:        func(repository.Repository, settings.LabelChange) error { writes++; return nil },
						UpdateLabel:        func(repository.Repository, settings.LabelChange) error { writes++; return nil },
						CreateLabel:        func(repository.Repository, settings.LabelChange) (int64, error) { writes++; return 0, nil },
					}
					result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w, prune)
					success := mode == "omitted" || mode == "empty" || mode == "normal-empty" || (mode == "nonempty" && tc.supportedNonempty)
					if (err == nil) != success || writes != 0 {
						t.Fatalf("outcome=%v err=%v writes=%d", result.Outcome, err, writes)
					}
					if !reflect.DeepEqual(d, before) {
						t.Fatal("declaration was changed")
					}
					if success && mode != "normal-empty" && reads != 2 {
						t.Fatalf("expected two identity reads, got %d", reads)
					}
					if mode == "normal-empty" && (reads != 0 || repositoryCalls != 1 || labelReads != 0) {
						t.Fatalf("ordinary apply did not skip empty collection: archive=%d repository=%d labels=%d", reads, repositoryCalls, labelReads)
					}
					if mode == "nonempty" && !tc.supportedNonempty && !strings.Contains(err.Error(), "without explicit archived: false") {
						t.Fatalf("nonempty request failed before archived guard: %v", err)
					}
					if success && !tc.supportedNonempty && labelReads != 0 {
						t.Fatalf("unselected label read: %d", labelReads)
					}
				})
			}
		})
	}
}

func TestArchivedEmptyEnvironmentVariableParentRetainsPruneIdentity(t *testing.T) {
	vars, err := settings.NewVariableDeclarations(nil)
	if err != nil {
		t.Fatal(err)
	}
	yes, description := true, "same"
	d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes, Description: &description}, EnvironmentVariables: []settings.EnvironmentVariableDeclaration{{Environment: "production", Variables: vars}}}
	before := settings.EnvironmentRetainedNames(d)
	if !slices.Contains(before, "production") {
		t.Fatal("variable-only parent omitted from retain set")
	}
	for _, scope := range []string{"environments", "environment-variables"} {
		prune, err := settings.ParsePruneScopes(scope)
		if err != nil {
			t.Fatal(err)
		}
		if archivedMatchingScope(d, prune) {
			t.Fatalf("prune %s was accepted", scope)
		}
		if got := settings.EnvironmentRetainedNames(d); !slices.Equal(got, before) {
			t.Fatalf("parent retain set changed: %v", got)
		}
	}
}

func TestArchivedAuthoritativeEmptyCollectionsStillRequireObservation(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	yes, desc := true, "same"
	emptyTopics := []string{}
	emptyExemptions, err := settings.NewPullRequestExemptionsDeclaration(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range []string{"topics", "exemptions"} {
		for _, scenario := range []string{"match", "different", "missing", "denied", "prune"} {
			t.Run(family+"/"+scenario, func(t *testing.T) {
				d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes, Description: &desc}}
				if family == "topics" {
					d.Topics.Names = &emptyTopics
				} else {
					d.PullRequestExemptions = &emptyExemptions
				}
				prune := settings.PruneScopes{}
				if scenario == "prune" {
					var e error
					prune, e = settings.ParsePruneScopes("webhooks")
					if e != nil {
						t.Fatal(e)
					}
				}
				reads, writes := 0, 0
				w := Writer{
					ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
						return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
					},
					ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
						return settings.NewRepositoryObservation(&settings.RepositoryState{Description: desc}, nil, nil), nil
					},
					ReadTopics: func(repository.Repository) ([]string, error) {
						reads++
						switch scenario {
						case "different":
							return []string{"unexpected"}, nil
						case "missing":
							return nil, nil
						case "denied":
							return nil, errors.New("denied")
						}
						return []string{}, nil
					},
					ReadPullRequestExemptions: func(repository.Repository) (settings.PullRequestExemptionsObservation, error) {
						reads++
						switch scenario {
						case "different":
							return settings.NewPullRequestExemptionsObservation([]settings.PullRequestExemptionUser{{ID: 7, Login: "person", Type: "User"}})
						case "missing":
							return settings.PullRequestExemptionsObservation{}, nil
						case "denied":
							return settings.PullRequestExemptionsObservation{}, errors.New("denied")
						}
						return settings.NewPullRequestExemptionsObservation(nil)
					},
					Topics: func(repository.Repository, settings.TopicsDeclaration) (bool, error) { writes++; return false, nil },
					PullRequestExemptions: func(repository.Repository, settings.PullRequestExemptionsDeclaration) (bool, error) {
						writes++
						return false, nil
					},
					CreatedID:          func(string, string, int64) error { writes++; return nil },
					BindCollaboratorID: func(string, int64) error { writes++; return nil },
				}
				result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w, prune)
				if (err == nil) != (scenario == "match") || writes != 0 {
					t.Fatalf("outcome=%v err=%v writes=%d", result.Outcome, err, writes)
				}
				if scenario == "prune" && reads != 0 {
					t.Fatalf("prune selector proceeded to comparison: %d", reads)
				}
				if scenario != "prune" && reads != 1 {
					t.Fatalf("authoritative empty not observed: %d", reads)
				}
			})
		}
	}
}
