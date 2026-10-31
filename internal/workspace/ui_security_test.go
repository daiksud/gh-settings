package workspace

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIConfigSecurityPlacementAndPresence(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
    settings:
        general:
            releases:
                immutable_release: false
        advanced_security:
            private_vulnerability_reporting: true
            secret_protection:
                enabled: true
                push_protection: false
`
	want := uimodel.Repository{Settings: &uimodel.RepositorySettings{
		General:            &uimodel.General{Releases: &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: uiConfigPtr(false)}},
		SecurityAndQuality: &uimodel.SecurityAndQuality{AdvancedSecurity: &uimodel.AdvancedSecurity{SecretProtection: uiConfigPtr(true), PushProtection: uiConfigPtr(false), PrivateVulnerabilityReporting: uiConfigPtr(true)}},
	}}
	got, err := DecodeConfig([]byte(input))
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Security paths changed: %#v, %v", got, err)
	}
	encoded, err := EncodeConfig(got)
	if err != nil || string(encoded) != input {
		t.Fatalf("Security golden: %s, %v", encoded, err)
	}
	got, err = DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {advanced_security: {secret_protection: {push_protection: false}}}}\n"))
	if err != nil || got.Settings.General != nil || got.Settings.SecurityAndQuality.AdvancedSecurity.SecretProtection != nil {
		t.Fatalf("omitted Security controls became intent: %#v, %v", got, err)
	}
}

func TestUIConfigSecurityRejectsInvalidPlacementAndTypes(t *testing.T) {
	for _, settings := range []string{
		"security: {}", "advanced_security: {general_releases: {}}", "general: {general_releases: {}}",
		"advanced_security: {secret_scanning: true}", "advanced_security: null",
		"advanced_security: {secret_protection: {push_protection: 'false'}}", "general: {releases: {immutable_release: null}}",
		"advanced_security: {enforced_by_owner: true}", "advanced_security: {secret_protection: null}",
		"advanced_security: {secret_protection: {enabled: 'true'}}",
	} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {" + settings + "}}\n")); err == nil {
			t.Fatalf("accepted invalid Security syntax: %s", settings)
		}
	}
}
