package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func checkoutConfigFixture(t *testing.T, input string) (workspace.Checkout, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "init", "--quiet", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	path := filepath.Join(root, workspace.SettingsPath)
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	return workspace.NewCheckout(filepath.Join(root, "nested")), path
}

const mixedCheckoutConfig = `version: daiksud/gh-settings@v0
repository:
  about: {description: '', website: 'https://example.com', topics: []}
  labels: [{name: new, color: '#abcdef', description: ''}]
  settings:
    general: {features: {issues: false}}
    actions:
      general:
        workflow_permissions: read_repository_contents_and_packages_permissions
        allow_github_actions_to_create_and_approve_pull_requests: false
`

func TestCheckoutUICompositionAndSnapshotWriteback(t *testing.T) {
	checkout, path := checkoutConfigFixture(t, mixedCheckoutConfig)
	empty, website, no, permission := "", "https://example.com", false, "read"
	color := "abcdef"
	labels, err := settings.NewLabelDeclarations([]settings.LabelDeclaration{{Name: "new", Color: &color, Description: &empty}})
	if err != nil {
		t.Fatal(err)
	}
	topics := []string{}
	want := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &website, HasIssues: &no}, Topics: settings.TopicsDeclaration{Names: &topics}, Labels: &labels, Actions: settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &permission, CanApprovePullRequestReviews: &no}}
	got, err := loadCheckoutDeclaration(checkout)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("checkout declaration %#v; want %#v; error=%v", got, want, err)
	}
	got, record, err := prepareCheckoutApply(checkout)
	if err != nil || record == nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot declaration %#v, recorder=%t, error=%v", got, record != nil, err)
	}
	if err := record("labels", "new", 17); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ui, err := workspace.DecodeConfig(data)
	if err != nil || ui.Labels == nil || len(*ui.Labels) != 1 || (*ui.Labels)[0].ID == nil || *(*ui.Labels)[0].ID != 17 {
		t.Fatalf("confirmed ID missing from UI snapshot: %s, %v", data, err)
	}
}

func TestCheckoutPublicCodecRejectsObsoleteEnvelope(t *testing.T) {
	if _, err := workspace.DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {has_issues: false}}\n")); err == nil {
		t.Fatal("public codec still accepts the obsolete API-shaped envelope")
	}
}

func TestCheckoutUICompositionNoOpDocuments(t *testing.T) {
	for _, input := range []string{"version: daiksud/gh-settings@v0\n", "version: daiksud/gh-settings@v0\nrepository: {}\n", "version: daiksud/gh-settings@v0\nrepository: {settings: {}}\n"} {
		checkout, _ := checkoutConfigFixture(t, input)
		got, err := loadCheckoutDeclaration(checkout)
		if err != nil || !got.Empty() {
			t.Fatalf("no-op document became intent: %#v, %v", got, err)
		}
	}
}

func TestCheckoutUIExportSaveDefersAndRejectsPartialOutput(t *testing.T) {
	checkout, path := checkoutConfigFixture(t, "# original configuration\n")
	save, err := prepareCheckoutSave(checkout)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil || string(before) != "# original configuration\n" {
		t.Fatalf("preparation changed destination: %s, %v", before, err)
	}
	if err := save(settings.Declaration{}); err == nil {
		t.Fatal("incomplete export was saved")
	}
	if after, err := os.ReadFile(path); err != nil || !bytes.Equal(after, before) {
		t.Fatalf("failed export changed destination: %s, %v", after, err)
	}
	empty, no := "", false
	topics := []string{}
	if err := save(settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &empty, HasIssues: &no}, Topics: settings.TopicsDeclaration{Names: &topics}}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(after), "version: daiksud/gh-settings@v0") || !strings.Contains(string(after), "features:\n                issues: false") || strings.Contains(string(after), "schemaVersion") || strings.Contains(string(after), "has_issues") {
		t.Fatalf("export did not save current UI YAML: %s, %v", after, err)
	}
}

func TestCheckoutUIInvalidDomainReturnsNoPartialIntent(t *testing.T) {
	checkout, path := checkoutConfigFixture(t, "version: daiksud/gh-settings@v0\nrepository: {about: {description: valid}, settings: {actions: {general: {workflow_permissions: read}}}}\n")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadCheckoutDeclaration(checkout)
	if err == nil || !got.Empty() {
		t.Fatalf("invalid UI choice returned partial intent: %#v, %v", got, err)
	}
	got, record, err := prepareCheckoutApply(checkout)
	if err == nil || !got.Empty() || record != nil {
		t.Fatalf("invalid snapshot returned intent/recorder: %#v, %t, %v", got, record != nil, err)
	}
	if after, err := os.ReadFile(path); err != nil || !bytes.Equal(after, before) {
		t.Fatalf("invalid intent changed source: %s, %v", after, err)
	}
}
