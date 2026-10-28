package github

import (
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"testing"
)

func TestUICreationPolicyRequiresVisibleFeature(t *testing.T) {
	for _, issues := range []bool{false, true} {
		for _, choice := range []uimodel.CreationAllowedBy{uimodel.AllUsers, uimodel.CollaboratorsOnly} {
			for _, enabled := range []*bool{nil, pointer(true), pointer(false)} {
				ui := uimodel.General{}
				if issues {
					ui.Features.Issues = enabled
					ui.Features.IssuePermissions.CreationAllowedBy = &choice
				} else {
					ui.Features.PullRequests = enabled
					ui.Features.PullRequestPermissions.CreationAllowedBy = &choice
				}
				_, err := GeneralDeclaration(ui)
				wantError := enabled != nil && !*enabled
				if (err != nil) != wantError {
					t.Fatalf("issues=%t feature=%v choice=%s err=%v", issues, enabled, choice, err)
				}
			}
		}
	}
	if _, err := GeneralDeclaration(uimodel.General{Features: uimodel.Features{Issues: pointer(false), PullRequests: pointer(false)}}); err != nil {
		t.Fatal("omitted policies became managed", err)
	}
}
