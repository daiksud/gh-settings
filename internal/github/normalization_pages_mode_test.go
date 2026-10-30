package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPagesModeValueProjectionKeepsScalarProvenance(t *testing.T) {
	for _, raw := range []string{"legacy", "workflow", "FUTURE_MODE", "", " \t"} {
		value := raw
		got, err := preparePagesModeValue(&value)
		wantMode := raw
		wantRecord := settings.ObservationNormalization{}
		if raw != "legacy" && raw != "workflow" {
			wantMode = "legacy"
			wantRecord = settings.ObservationNormalization{Path: "repository.settings.pages.build_and_deployment.source", RawValue: raw, Fallback: "deploy_from_a_branch"}
		}
		if err != nil || got.BuildType != wantMode || got.Normalization != wantRecord {
			t.Fatal(raw, got, wantMode, wantRecord, err)
		}
		value = "mutated caller"
		if got.BuildType != wantMode || got.Normalization != wantRecord {
			t.Fatal("aliased scalar", got)
		}
	}
	if got, err := preparePagesModeValue(nil); err == nil || got.BuildType != "" || got.Normalization != (settings.ObservationNormalization{}) {
		t.Fatal("missing mode invented", got, err)
	}
}
