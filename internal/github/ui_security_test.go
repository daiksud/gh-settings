package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
)

func securityUIPtr[T any](value T) *T { return &value }
func TestUISecurityScanningControls(t *testing.T) {
	if got, err := SecurityDeclaration(uimodel.SecuritySettings{}); err != nil || !reflect.DeepEqual(got, settings.SecurityDeclaration{}) {
		t.Fatal(got, err)
	}
	for _, v := range []bool{false, true} {
		for i := 0; i < 2; i++ {
			control := uimodel.AdvancedSecurity{}
			var want settings.SecurityDeclaration
			if i == 0 {
				control.SecretProtection = &v
				want.SecretScanning = &v
			} else {
				control.PushProtection = &v
				want.PushProtection = &v
			}
			got, err := SecurityDeclaration(uimodel.SecuritySettings{AdvancedSecurity: &control})
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatal(got, err)
			}
		}
	}
	if _, err := SecurityDeclaration(uimodel.SecuritySettings{AdvancedSecurity: &uimodel.AdvancedSecurity{SecretProtection: securityUIPtr(false), PushProtection: securityUIPtr(true)}}); err == nil {
		t.Fatal("inconsistent desired controls accepted")
	}
	value := true
	got, err := SecurityDeclaration(uimodel.SecuritySettings{AdvancedSecurity: &uimodel.AdvancedSecurity{SecretProtection: &value}})
	if err != nil {
		t.Fatal(err)
	}
	value = false
	if !*got.SecretScanning {
		t.Fatal("desired value aliases caller")
	}
	for _, v := range []bool{false, true} {
		view, err := WithSecurityScanningObservation(uimodel.SecuritySettings{}, settings.NewSecretScanningObservation(v, v))
		if err != nil || view.AdvancedSecurity == nil || *view.AdvancedSecurity.SecretProtection != v || *view.AdvancedSecurity.PushProtection != v {
			t.Fatal(view, err)
		}
	}
	if _, err := WithSecurityScanningObservation(uimodel.SecuritySettings{}, settings.SecretScanningObservation{}); err == nil {
		t.Fatal("unacquired state defaulted")
	}
	// A complete inconsistent source is an observation, not a valid declaration.
	raw, err := WithSecurityScanningObservation(uimodel.SecuritySettings{}, settings.NewSecretScanningObservation(false, true))
	if err != nil || *raw.AdvancedSecurity.SecretProtection || !*raw.AdvancedSecurity.PushProtection {
		t.Fatal(raw, err)
	}
	if _, err := SecurityDeclaration(raw); err == nil {
		t.Fatal("raw inconsistency became desired state")
	}
}
