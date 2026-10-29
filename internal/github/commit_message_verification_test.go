package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestRepositoryCommitMessagesMatchRawEvidence(t *testing.T) {
	for _, menu := range []settings.CommitMessageMenu{settings.SquashCommitMenu, settings.MergeCommitMenu} {
		t.Run(string(menu), func(t *testing.T) {
			fallbackTitle, fallbackMessage := "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES"
			if menu == settings.MergeCommitMenu {
				fallbackTitle, fallbackMessage = "MERGE_MESSAGE", "PR_TITLE"
			}
			observation := func(title, message string, normalized bool) repositoryCommitObservations {
				choice := uimodel.DefaultMessage
				pair := settings.CommitMessagePair{Title: title, Message: message}
				var provenance settings.CommitMessageNormalization
				if normalized {
					pair = settings.CommitMessagePair{Title: fallbackTitle, Message: fallbackMessage}
					provenance = settings.CommitMessageNormalization{Path: menu.Path(), RawTitle: title, RawMessage: message, Fallback: uimodel.DefaultMessage}
				} else if title == "PR_TITLE" && message == "BLANK" {
					choice = uimodel.PullRequestTitle
				}
				return repositoryCommitObservations{menu: {Choice: &choice, Pair: &pair, Normalization: provenance}}
			}
			intent := func(title, message *string) settings.RepositoryDeclaration {
				if menu == settings.SquashCommitMenu {
					return settings.RepositoryDeclaration{SquashMergeCommitTitle: title, SquashMergeCommitMessage: message}
				}
				return settings.RepositoryDeclaration{MergeCommitTitle: title, MergeCommitMessage: message}
			}
			before := observation("FUTURE_TITLE", "FUTURE_MESSAGE", true)
			title, message := "PR_TITLE", "BLANK"
			description := "scalar only"
			for _, test := range []struct {
				name    string
				actual  repositoryCommitObservations
				desired settings.RepositoryDeclaration
				want    bool
			}{
				{"equal fallback still unknown", before, intent(&fallbackTitle, &fallbackMessage), false},
				{"different desired still unknown", before, intent(&title, &message), false},
				{"different unknown raw", observation("OTHER_TITLE", "OTHER_MESSAGE", true), intent(&fallbackTitle, &fallbackMessage), false},
				{"canonical fallback", observation(fallbackTitle, fallbackMessage, false), intent(&fallbackTitle, &fallbackMessage), true},
				{"canonical different", observation(title, message, false), intent(&title, &message), true},
				{"missing observation", nil, intent(&title, &message), false},
				{"unmanaged", nil, settings.RepositoryDeclaration{}, true},
				{"unrelated scalar", repositoryCommitObservations{menu: {}}, settings.RepositoryDeclaration{Description: &description}, true},
				{"partial preserves raw companion", observation(title, "FUTURE_MESSAGE", true), intent(&title, nil), true},
				{"partial companion changed", observation(title, "OTHER_MESSAGE", true), intent(&title, nil), false},
				{"message preserves raw title", observation("FUTURE_TITLE", message, true), intent(nil, &message), true},
				{"message raw title changed", observation("OTHER_TITLE", message, true), intent(nil, &message), false},
			} {
				t.Run(test.name, func(t *testing.T) {
					if got := repositoryCommitMessagesMatch(before, test.actual, test.desired); got != test.want {
						t.Fatalf("match = %v, want %v", got, test.want)
					}
				})
			}
			for _, corrupt := range []func(*preparedCommitMessage){
				func(p *preparedCommitMessage) { p.Choice = nil },
				func(p *preparedCommitMessage) { p.Pair = nil },
				func(p *preparedCommitMessage) { p.Normalization.Path = "wrong" },
				func(p *preparedCommitMessage) { p.Normalization.Fallback = uimodel.PullRequestTitle },
				func(p *preparedCommitMessage) { p.Pair.Title = "wrong" },
				func(p *preparedCommitMessage) { *p.Choice = uimodel.PullRequestTitle },
				func(p *preparedCommitMessage) {
					p.Normalization.RawTitle, p.Normalization.RawMessage = fallbackTitle, fallbackMessage
				},
			} {
				actual := observation(title, "FUTURE_MESSAGE", true)
				value := actual[menu]
				corrupt(&value)
				actual[menu] = value
				if repositoryCommitMessagesMatch(before, actual, intent(&title, nil)) {
					t.Fatal("inconsistent acquired provenance manufactured a match")
				}
			}
		})
	}
}
