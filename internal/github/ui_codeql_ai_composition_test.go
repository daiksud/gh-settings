package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestCodeQLConfigurationAndAIScanExportRemainIndependent(t *testing.T) {
	ai, suite := false, "extended"
	source := uiExportBase()
	source.Security = settings.SecurityDeclaration{
		AIScanPullRequests:  &ai,
		CodeQLConfiguration: settings.CodeQLConfigurationDeclaration{QuerySuite: &suite},
	}
	ui, err := UIFromExportDeclaration(source)
	if err != nil || ui.Settings == nil || ui.Settings.SecurityAndQuality == nil || ui.Settings.SecurityAndQuality.AdvancedSecurity == nil || ui.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity == nil {
		t.Fatalf("combined export unavailable: %#v, %v", ui, err)
	}
	cs := ui.Settings.SecurityAndQuality.AdvancedSecurity.CodeSecurity
	if cs.AIScan == nil || cs.AIScan.AIScanForPullRequests == nil || *cs.AIScan.AIScanForPullRequests || cs.CodeQLAnalysis == nil || cs.CodeQLAnalysis.QuerySuite == nil || *cs.CodeQLAnalysis.QuerySuite != "extended" || cs.CodeQLAnalysis.DefaultSetup != nil || cs.CodeQLAnalysis.Runner != nil || cs.CodeQLAnalysis.Languages != nil {
		t.Fatalf("combined export lost or expanded intent: %#v", cs)
	}
	native, err := DeclarationFromUI(ui)
	if err != nil || native.Security.AIScanPullRequests == nil || *native.Security.AIScanPullRequests || native.Security.CodeQLConfiguration.QuerySuite == nil || *native.Security.CodeQLConfiguration.QuerySuite != "extended" || native.Security.CodeQLDefaultSetup != nil || native.Security.PrivateVulnerabilityReporting != nil || native.Security.CodeQLConfiguration.Runner != nil || native.Security.CodeQLConfiguration.Languages != nil {
		t.Fatalf("combined round trip changed ownership: %#v, %v", native.Security, err)
	}
}
