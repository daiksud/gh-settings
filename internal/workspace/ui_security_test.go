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
                enable_release_immutability: false
        security_and_quality:
            advanced_security:
                secret_protection: true
                push_protection: false
                private_vulnerability_reporting: true
`
	want := uimodel.Repository{Settings: &uimodel.RepositorySettings{
		General:            &uimodel.General{Releases: &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: uiConfigPtr(false)}},
		SecurityAndQuality: &uimodel.SecurityAndQuality{AdvancedSecurity: &uimodel.AdvancedSecurity{SecretProtection: uiConfigPtr(true), PushProtection: uiConfigPtr(false), PrivateVulnerabilityReporting: uiConfigPtr(true)}},
	}}
	got, err := DecodeUIConfig([]byte(input))
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Security paths changed: %#v, %v", got, err)
	}
	encoded, err := EncodeUIConfig(got)
	if err != nil || string(encoded) != input {
		t.Fatalf("Security golden: %s, %v", encoded, err)
	}
	got, err = DecodeUIConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {security_and_quality: {advanced_security: {push_protection: false}}}}\n"))
	if err != nil || got.Settings.General != nil || got.Settings.SecurityAndQuality.AdvancedSecurity.SecretProtection != nil {
		t.Fatalf("omitted Security controls became intent: %#v, %v", got, err)
	}
}

func TestUIConfigSecurityRejectsInvalidPlacementAndTypes(t *testing.T) {
	for _, settings := range []string{
		"security: {}", "security_and_quality: {general_releases: {}}", "general: {general_releases: {}}",
		"security_and_quality: {advanced_security: {secret_scanning: true}}", "security_and_quality: {advanced_security: null}",
		"security_and_quality: {advanced_security: {push_protection: 'false'}}", "general: {releases: {enable_release_immutability: null}}",
		"security_and_quality: {enforced_by_owner: true}",
	} {
		if _, err := DecodeUIConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {" + settings + "}}\n")); err == nil {
			t.Fatalf("accepted invalid Security syntax: %s", settings)
		}
	}
}
