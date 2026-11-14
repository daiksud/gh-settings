package app

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// Export may retain prior intent for unavailable families. This fixture checks
// that neither retained declarations nor a visible null Copilot override are
// promoted to proof of remote equality by archived apply.
func TestArchivedMixedExportFailsClosedWithoutCompleteEvidence(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	resolve := func() (repository.Repository, error) { return repo, nil }
	reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main", Description: "example", HasIssues: true}, func(string, repository.Repository) error { return nil })
	reader.Archive = func(repository.Repository) (settings.ArchiveState, error) {
		return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
	}
	reader.CopilotReviewEffort = func(repository.Repository) (settings.CopilotReviewEffortObservation, error) {
		return settings.NewCopilotReviewEffortObservation(nil, "balanced", []string{"lite", "balanced"})
	}
	var exported settings.Declaration
	err := Export(ExportCheckout{Existing: exportPrior(t), Repository: resolve, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(d settings.Declaration) error { exported = d; return nil }, nil
	}}, reader)
	if err != nil {
		t.Fatal(err)
	}
	ui, err := github.UIFromExportDeclaration(exported)
	if err != nil {
		t.Fatal(err)
	}
	data, err := workspace.EncodeConfig(ui)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := workspace.DecodeConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := github.DeclarationFromUI(decoded)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err = Validate(func() (settings.Declaration, error) { return declaration, nil })
	if err != nil {
		t.Fatal(err)
	}
	if declaration.CopilotReviewEffort == nil || *declaration.CopilotReviewEffort != "default" {
		t.Fatal("fixture did not export the unverified default")
	}
	if declaration.Repository.Archived == nil || !*declaration.Repository.Archived {
		t.Fatal("fixture did not export archived state")
	}
	if archivedMatchingScope(declaration, settings.PruneScopes{}) {
		t.Fatal("mixed export with unsupported declarations appeared fully covered")
	}
	writes := 0
	archiveReader := func(repository.Repository) (settings.ArchiveState, error) {
		return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
	}
	writer := Writer{ReadArchive: archiveReader, Archive: func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil }, CreatedID: func(string, string, int64) error { writes++; return nil }}
	_, err = Apply(func() (settings.Declaration, error) { return declaration, nil }, resolve, writer)
	if err == nil || writes != 0 {
		t.Fatalf("mixed export accepted or mutated: error=%v writes=%d", err, writes)
	}
	// Restrict to the exported null/default value to check its uncertainty even
	// after every other unsupported declaration has been removed.
	limited := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: declaration.Repository.Archived}, CopilotReviewEffort: declaration.CopilotReviewEffort}
	writer.ReadCopilotReviewEffort = reader.CopilotReviewEffort
	_, err = Apply(func() (settings.Declaration, error) { return limited, nil }, resolve, writer)
	if err == nil || writes != 0 {
		t.Fatalf("unverified null/default accepted or mutated: error=%v writes=%d", err, writes)
	}
}
