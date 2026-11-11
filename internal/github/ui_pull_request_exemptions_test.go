package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestPullRequestExemptionsExportMergesIndependentCapControls(t *testing.T) {
	for _, withCap := range []bool{false, true} {
		empty := ""
		topics := []string{}
		enabled := false
		maximum := 10
		drafts := true
		exemptions, _ := settings.NewPullRequestExemptionsDeclaration([]string{})
		d := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &empty}, Topics: settings.TopicsDeclaration{Names: &topics}, PullRequestExemptions: &exemptions}
		if withCap {
			d.PullRequestLimits = settings.PullRequestLimitsDeclaration{Enabled: &enabled, Maximum: &maximum, IncludeDrafts: &drafts}
		}
		ui, err := UIFromExportDeclaration(d)
		if err != nil {
			t.Fatal(err)
		}
		control := ui.Settings.InteractionLimits.PullRequestLimits
		if control.ExemptionList == nil || len(*control.ExemptionList) != 0 || (control.MaximumOpenPullRequestsPerUser != nil) != withCap {
			t.Fatalf("export ownership %#v", control)
		}
		again, err := DeclarationFromUI(ui)
		if err != nil || again.PullRequestExemptions == nil || again.PullRequestLimits.Empty() == withCap {
			t.Fatalf("reimport %#v %v", again, err)
		}
	}
}
