package github

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestUISecurityImmutableControlAndMetadata(t *testing.T) {
	for _, value := range []bool{false, true} {
		got, err := SecurityDeclaration(uimodel.SecuritySettings{GeneralReleases: &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: &value}})
		if err != nil || !reflect.DeepEqual(got, settings.SecurityDeclaration{ImmutableReleases: &value}) {
			t.Fatal(got, err)
		}
	}
	for _, test := range []struct {
		o       settings.ImmutableReleasesObservation
		enabled bool
		owner   *bool
	}{
		{settings.NewImmutableReleasesObservation(true, true), true, securityUIPtr(true)},
		{settings.NewImmutableReleasesObservation(true, false), true, securityUIPtr(false)},
		{settings.NewImmutableReleasesObservation(false, false), false, securityUIPtr(false)},
		{settings.NewDisabledImmutableReleasesObservation(), false, nil},
	} {
		base := uimodel.SecuritySettings{AdvancedSecurity: &uimodel.AdvancedSecurity{SecretProtection: securityUIPtr(false)}, GeneralReleases: &uimodel.SecurityGeneralReleases{}}
		view, err := WithSecurityImmutableObservation(base, test.o)
		if err != nil || *view.Settings.GeneralReleases.EnableReleaseImmutability != test.enabled || !reflect.DeepEqual(view.EnforcedByOwner, test.owner) || view.Settings.AdvancedSecurity.SecretProtection == nil || base.GeneralReleases.EnableReleaseImmutability != nil {
			t.Fatal(view, base, err)
		}
	}
	for _, o := range []settings.ImmutableReleasesObservation{{}, settings.NewImmutableReleasesObservation(false, true)} {
		if _, err := WithSecurityImmutableObservation(uimodel.SecuritySettings{}, o); err == nil {
			t.Fatal("invalid/unacquired state accepted")
		}
	}
	d, err := SecurityDeclaration(uimodel.SecuritySettings{GeneralReleases: &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: securityUIPtr(false)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := settings.CompareImmutableReleases(settings.NewImmutableReleasesObservation(true, true), d); err == nil {
		t.Fatal("owner enforcement lost")
	}
}
func TestUISecurityImmutableExactToggleAnd404(t *testing.T) {
	for _, value := range []bool{false, true} {
		calls := 0
		written := false
		client := &ImmutableReleasesClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Path != "/repos/owner/current/immutable-releases" {
				t.Fatal(req.URL)
			}
			if req.Method == "PUT" || req.Method == "DELETE" {
				want := "DELETE"
				if value {
					want = "PUT"
				}
				if req.Method != want {
					t.Fatal(req.Method)
				}
				if req.Body != nil {
					body, _ := io.ReadAll(req.Body)
					if len(body) != 0 {
						t.Fatal(string(body))
					}
				}
				written = true
				return labelsResponse(req, 204, ""), nil
			}
			if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			enabled := !value
			if written {
				enabled = value
			}
			if !enabled {
				return labelsResponse(req, 404, `{"message":"Not Found"}`), nil
			}
			return labelsResponse(req, 200, `{"enabled":true,"enforced_by_owner":false}`), nil
		}).rest}
		repo := repository.Repository{Owner: "owner", Name: "current"}
		initial, err := client.Read(repo)
		if err != nil {
			t.Fatal(err)
		}
		projected, err := WithSecurityImmutableObservation(uimodel.SecuritySettings{}, initial)
		if err != nil || *projected.Settings.GeneralReleases.EnableReleaseImmutability == value {
			t.Fatal(projected, err)
		}
		d, err := SecurityDeclaration(uimodel.SecuritySettings{GeneralReleases: &uimodel.SecurityGeneralReleases{EnableReleaseImmutability: &value}})
		if err != nil {
			t.Fatal(err)
		}
		_, changes, err := settings.CompareImmutableReleases(initial, d)
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		changed, err := client.Update(repo, changes[0])
		if err != nil || !changed || calls != 4 {
			t.Fatal(changed, calls, err)
		}
	}
}
