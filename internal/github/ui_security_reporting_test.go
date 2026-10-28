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

func TestUISecurityReportingControl(t *testing.T) {
	for _, value := range []bool{false, true} {
		got, err := SecurityDeclaration(uimodel.SecuritySettings{AdvancedSecurity: &uimodel.AdvancedSecurity{PrivateVulnerabilityReporting: &value}})
		if err != nil || !reflect.DeepEqual(got, settings.SecurityDeclaration{PrivateVulnerabilityReporting: &value}) {
			t.Fatal(got, err)
		}
		base := uimodel.SecuritySettings{AdvancedSecurity: &uimodel.AdvancedSecurity{SecretProtection: securityUIPtr(false)}}
		view, err := WithSecurityReportingObservation(base, settings.NewPrivateReportingObservation(value))
		if err != nil || view.AdvancedSecurity.PrivateVulnerabilityReporting == nil || *view.AdvancedSecurity.PrivateVulnerabilityReporting != value || view.AdvancedSecurity.SecretProtection == nil || *view.AdvancedSecurity.SecretProtection || base.AdvancedSecurity.PrivateVulnerabilityReporting != nil {
			t.Fatal(view, base, err)
		}
	}
	if _, err := WithSecurityReportingObservation(uimodel.SecuritySettings{}, settings.PrivateReportingObservation{}); err == nil {
		t.Fatal("unacquired defaulted")
	}
}
func TestUISecurityReportingExactToggle(t *testing.T) {
	for _, value := range []bool{false, true} {
		calls := 0
		written := false
		client := &PrivateReportingClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Path == "/repos/owner/current" && req.Method == "GET" {
				return labelsResponse(req, 200, `{"visibility":"public"}`), nil
			}
			if req.URL.Path != "/repos/owner/current/private-vulnerability-reporting" {
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
					if len(body) > 0 {
						t.Fatal(string(body))
					}
				}
				written = true
				return labelsResponse(req, 204, ""), nil
			}
			if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			state := !value
			if written {
				state = value
			}
			body := `{"enabled":false}`
			if state {
				body = `{"enabled":true}`
			}
			return labelsResponse(req, 200, body), nil
		}).rest}
		d, err := SecurityDeclaration(uimodel.SecuritySettings{AdvancedSecurity: &uimodel.AdvancedSecurity{PrivateVulnerabilityReporting: &value}})
		if err != nil {
			t.Fatal(err)
		}
		_, changes, err := settings.ComparePrivateReporting(settings.NewPrivateReportingObservation(!value), d)
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		changed, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, changes[0])
		if err != nil || !changed || calls != 5 {
			t.Fatal(changed, calls, err)
		}
	}
}
