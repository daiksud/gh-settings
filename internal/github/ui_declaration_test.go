package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestDeclarationFromUIAboutAndGeneral(t *testing.T) {
	empty, website, branch := "", "https://example.com", "main"
	yes, no := true, false
	choice := uimodel.PullRequestTitle
	topics := []string{"settings", "github"}
	ui := uimodel.Repository{
		About: &uimodel.About{Description: &empty, Website: &website, Topics: &topics},
		Settings: &uimodel.RepositorySettings{General: &uimodel.General{
			TemplateRepository: &no, DefaultBranch: &branch,
			Features:     uimodel.Features{Issues: &yes},
			PullRequests: uimodel.PullRequests{AllowMergeCommits: &yes, MergeCommit: uimodel.CommitMessage{DefaultCommitMessage: &choice}},
		}},
	}
	title, message := "PR_TITLE", "BLANK"
	wantTopics := []string{"github", "settings"}
	want := settings.Declaration{
		Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &website, IsTemplate: &no, DefaultBranch: &branch, HasIssues: &yes, AllowMergeCommit: &yes, MergeCommitTitle: &title, MergeCommitMessage: &message},
		Topics:     settings.TopicsDeclaration{Names: &wantTopics},
	}
	got, err := DeclarationFromUI(ui)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("declaration %#v, error %v; want %#v", got, err, want)
	}
}

func TestDeclarationFromUIEmptyGroups(t *testing.T) {
	for _, ui := range []uimodel.Repository{{}, {About: &uimodel.About{}}, {Settings: &uimodel.RepositorySettings{General: &uimodel.General{}}}} {
		got, err := DeclarationFromUI(ui)
		if err != nil || !got.Empty() {
			t.Fatalf("omitted controls became intent: %#v, %v", got, err)
		}
	}
}

func TestDeclarationFromUIRejectsInvalidChoiceAndDependencies(t *testing.T) {
	validAbout := "preserve only on success"
	apiChoice := uimodel.CommitMessageChoice("PR_TITLE")
	creation := uimodel.AllUsers
	no := false
	for _, general := range []uimodel.General{
		{PullRequests: uimodel.PullRequests{MergeCommit: uimodel.CommitMessage{DefaultCommitMessage: &apiChoice}}},
		{Features: uimodel.Features{Issues: &no, IssuePermissions: uimodel.CreationPermissions{CreationAllowedBy: &creation}}},
	} {
		got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &validAbout}, Settings: &uimodel.RepositorySettings{General: &general}})
		if err == nil || !got.Empty() {
			t.Fatalf("accepted invalid UI intent or returned partial declaration: %#v, %v", got, err)
		}
	}
}

func TestDeclarationFromUILabelIdentityAndColor(t *testing.T) {
	id, selectedColor, newColor, empty := int64(17), "#ABCDEF", "#123456", ""
	labels := []uimodel.Label{{ID: &id, Name: "renamed", Color: &selectedColor, Description: &empty}, {Name: "new", Color: &newColor}}
	got, err := DeclarationFromUI(uimodel.Repository{Labels: &labels})
	if err != nil || got.Labels == nil {
		t.Fatalf("missing converted labels: %v", err)
	}
	values := got.Labels.Values()
	if len(values) != 2 || values[0].ID == nil || *values[0].ID != 17 || values[0].Name != "renamed" || values[0].Color == nil || *values[0].Color != "abcdef" || values[0].Description == nil || *values[0].Description != "" || values[1].ID != nil || values[1].Color == nil || *values[1].Color != "123456" || values[1].Description != nil {
		t.Fatalf("label conversion changed identity/presence: %#v", values)
	}
	if *labels[0].Color != "#ABCDEF" {
		t.Fatal("conversion modified UI intent")
	}
}

func TestDeclarationFromUILabelCollectionPresence(t *testing.T) {
	empty := []uimodel.Label{}
	for _, labels := range []*[]uimodel.Label{nil, &empty} {
		got, err := DeclarationFromUI(uimodel.Repository{Labels: labels})
		if err != nil || (got.Labels != nil) != (labels != nil) || (got.Labels != nil && got.Labels.Len() != 0) {
			t.Fatalf("collection presence changed: %#v, %v", got.Labels, err)
		}
	}
}

func TestDeclarationFromUILabelInvalidIntentIsAtomic(t *testing.T) {
	id, color, noPrefix, description := int64(17), "#abcdef", "abcdef", "valid About"
	for _, labels := range [][]uimodel.Label{
		{{Name: "bad-color", Color: &noPrefix}},
		{{ID: &id, Name: "one", Color: &color}, {ID: &id, Name: "two", Color: &color}},
	} {
		got, err := DeclarationFromUI(uimodel.Repository{About: &uimodel.About{Description: &description}, Labels: &labels})
		if err == nil || !got.Empty() {
			t.Fatalf("accepted invalid labels or returned partial intent: empty=%v, error=%v", got.Empty(), err)
		}
	}
}
