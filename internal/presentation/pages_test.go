package presentation

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPagesNormalizedModeDisplaysUIChoicesAndQuotedRawWarning(t *testing.T) {
	record := settings.ObservationNormalization{Path: "repository.settings.pages.build_and_deployment.source", RawValue: "FUTURE\n<script>\"", Fallback: "deploy_from_a_branch"}
	for _, test := range []struct{ mode, choice string }{{"legacy", "deploy_from_a_branch"}, {"workflow", "github_actions"}} {
		change, err := settings.NewPagesModeChangeWithNormalization("pages.build_type", "legacy", test.mode, false, record)
		if err != nil {
			t.Fatal(err)
		}
		want := "Warning: repository.settings.pages.build_and_deployment.source: observed \"FUTURE\\n<script>\\\"\" is outside UI choices; using \"deploy_from_a_branch\" (projection only).\nrepository.settings.pages.build_and_deployment.source: \"deploy_from_a_branch\" (normalized observation) → \"" + test.choice + "\" (write required)"
		if test.mode == "workflow" {
			want += " (branch source becomes inactive)"
		}
		if got := PagesLine(change); got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	}
	folder, err := settings.NewPagesChangeWithNormalization("pages.source.path", "/", "/docs", false, settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: "/future", Fallback: "root"})
	if err != nil {
		t.Fatal(err)
	}
	wantFolder := "Warning: repository.settings.pages.build_and_deployment.select_folder: observed \"/future\" is outside UI choices; using \"root\" (projection only).\nrepository.settings.pages.build_and_deployment.select_folder: \"root\" (normalized observation) → \"docs\" (write required)"
	if got := PagesLine(folder); got != wantFolder {
		t.Fatal(got)
	}
	ordinary, err := settings.NewPagesChange("pages.build_type", "legacy", "workflow", false)
	if err != nil {
		t.Fatal(err)
	}
	if got := PagesLine(ordinary); got != "settings.pages.build_type: \"legacy\" → \"workflow\" (branch source becomes inactive)" {
		t.Fatal(got)
	}
}
