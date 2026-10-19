package app

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestApplyLabelSequenceAndFailureStop(t *testing.T) {
	for _, failAt := range []string{"", "create:a", "update:B", "delete:C"} {
		t.Run(failAt, func(t *testing.T) {
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			cause := errors.New("fixture failure")
			var calls []string
			step := func(name string, repo repository.Repository) error {
				if repo != target {
					t.Fatalf("target=%+v", repo)
				}
				calls = append(calls, name)
				if name == failAt {
					return cause
				}
				return nil
			}
			labelWrite := func(kind string) func(repository.Repository, settings.LabelChange) error {
				return func(repo repository.Repository, change settings.LabelChange) error {
					if change.Operation() != kind {
						t.Fatalf("wrong writer for %s", change.Operation())
					}
					return step(kind+":"+change.Name(), repo)
				}
			}
			color, disabled, names := "000000", false, []string{}
			labels := appLabelDeclarations(t, settings.LabelDeclaration{Name: "b", Color: &color}, settings.LabelDeclaration{Name: "a"})
			result, err := Apply(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}, Topics: settings.TopicsDeclaration{Names: &names}, Labels: &labels}, nil
			}, func() (repository.Repository, error) {
				calls = append(calls, "target")
				return target, nil
			}, Writer{
				Repository: func(repo repository.Repository, desired settings.RepositoryDeclaration) error {
					if desired.HasIssues == nil || *desired.HasIssues {
						t.Fatalf("repository=%+v", desired)
					}
					return step("repository", repo)
				},
				Topics: func(repo repository.Repository, desired settings.TopicsDeclaration) error {
					if desired.Names == nil || len(*desired.Names) != 0 {
						t.Fatalf("topics=%+v", desired)
					}
					return step("topics", repo)
				},
				ReadLabels: func(repo repository.Repository) (settings.LabelObservation, error) {
					if err := step("labels-read", repo); err != nil {
						return settings.LabelObservation{}, err
					}
					return settings.NewLabelObservation([]settings.LabelState{{Name: "C", Color: "abcdef"}, {Name: "B", Color: "abcdef", Description: "keep"}})
				},
				CreateLabel: labelWrite("create"), UpdateLabel: labelWrite("update"), DeleteLabel: labelWrite("delete"),
			})
			want := []string{"load", "target", "repository", "topics", "labels-read", "create:a", "update:B", "delete:C"}
			if failAt != "" {
				for i, name := range want {
					if name == failAt {
						want = want[:i+1]
						break
					}
				}
				kind, name, _ := strings.Cut(failAt, ":")
				message := fmt.Sprintf("%s label %q (earlier successful writes remain applied): fixture failure", kind, name)
				if !errors.Is(err, cause) || err.Error() != message || result.Completed() {
					t.Fatalf("result=%+v error=%v", result, err)
				}
			} else if err != nil || !result.Completed() {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("calls=%v want=%v", calls, want)
			}
		})
	}
}
