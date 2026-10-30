package github

import (
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"net/http"
	"reflect"
	"testing"
)

func TestUISecurityScanningExactRequestsAndRawReads(t *testing.T) {
	for _, test := range []struct {
		ui                     uimodel.AdvancedSecurity
		before, after, payload string
		beforeUI               [2]bool
	}{
		{uimodel.AdvancedSecurity{SecretProtection: securityUIPtr(true)}, `{"security_and_analysis":{"secret_scanning":{"status":"disabled"},"secret_scanning_push_protection":{"status":"disabled"}}}`, `{"security_and_analysis":{"secret_scanning":{"status":"enabled"},"secret_scanning_push_protection":{"status":"disabled"}}}`, `{"security_and_analysis":{"secret_scanning":{"status":"enabled"}}}`, [2]bool{false, false}},
		{uimodel.AdvancedSecurity{SecretProtection: securityUIPtr(false)}, `{"security_and_analysis":{"secret_scanning":{"status":"enabled"},"secret_scanning_push_protection":{"status":"disabled"}}}`, `{"security_and_analysis":{"secret_scanning":{"status":"disabled"},"secret_scanning_push_protection":{"status":"disabled"}}}`, `{"security_and_analysis":{"secret_scanning":{"status":"disabled"}}}`, [2]bool{true, false}},
		{uimodel.AdvancedSecurity{PushProtection: securityUIPtr(true)}, `{"security_and_analysis":{"secret_scanning":{"status":"enabled"},"secret_scanning_push_protection":{"status":"disabled"}}}`, `{"security_and_analysis":{"secret_scanning":{"status":"enabled"},"secret_scanning_push_protection":{"status":"enabled"}}}`, `{"security_and_analysis":{"secret_scanning_push_protection":{"status":"enabled"}}}`, [2]bool{true, false}},
		{uimodel.AdvancedSecurity{PushProtection: securityUIPtr(false)}, `{"security_and_analysis":{"secret_scanning":{"status":"enabled"},"secret_scanning_push_protection":{"status":"enabled"}}}`, `{"security_and_analysis":{"secret_scanning":{"status":"enabled"},"secret_scanning_push_protection":{"status":"disabled"}}}`, `{"security_and_analysis":{"secret_scanning_push_protection":{"status":"disabled"}}}`, [2]bool{true, true}},
	} {
		calls := 0
		written := false
		client := &SecretScanningClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Path != "/repos/owner/current" {
				t.Fatal(req.URL)
			}
			if req.Method == "PATCH" {
				var got, want any
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(test.payload), &want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatal(got, want)
				}
				written = true
				return labelsResponse(req, 200, `{}`), nil
			}
			if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			body := test.before
			if written {
				body = test.after
			}
			return labelsResponse(req, 200, body), nil
		}).rest}
		repo := repository.Repository{Owner: "owner", Name: "current"}
		initial, err := client.Read(repo)
		if err != nil {
			t.Fatal(err)
		}
		ui, err := WithSecurityScanningObservation(uimodel.SecuritySettings{}, initial)
		if err != nil || *ui.AdvancedSecurity.SecretProtection != test.beforeUI[0] || *ui.AdvancedSecurity.PushProtection != test.beforeUI[1] {
			t.Fatal(ui, err)
		}
		d, err := SecurityDeclaration(uimodel.SecuritySettings{AdvancedSecurity: &test.ui})
		if err != nil {
			t.Fatal(err)
		}
		changes, err := settings.SecretScanningTransitions(initial, d)
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		changed, err := client.Update(repo, changes[0])
		if err != nil || !changed || calls != 4 {
			t.Fatal(changed, calls, err)
		}
	}
}
func TestUISecurityDependenciesAndFailedAcquisition(t *testing.T) {
	for _, test := range []struct {
		ui      uimodel.AdvancedSecurity
		current settings.SecretScanningObservation
	}{
		{uimodel.AdvancedSecurity{PushProtection: securityUIPtr(true)}, settings.NewSecretScanningObservation(false, false)},
		{uimodel.AdvancedSecurity{SecretProtection: securityUIPtr(false)}, settings.NewSecretScanningObservation(true, true)},
	} {
		d, err := SecurityDeclaration(uimodel.SecuritySettings{AdvancedSecurity: &test.ui})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := settings.SecretScanningTransitions(test.current, d); err == nil {
			t.Fatal("companion defaulted")
		}
	}
	for _, enabled := range []bool{false, true} {
		d, err := SecurityDeclaration(uimodel.SecuritySettings{AdvancedSecurity: &uimodel.AdvancedSecurity{SecretProtection: &enabled, PushProtection: &enabled}})
		if err != nil {
			t.Fatal(err)
		}
		changes, err := settings.SecretScanningTransitions(settings.NewSecretScanningObservation(!enabled, !enabled), d)
		if err != nil || len(changes) != 2 {
			t.Fatal(changes, err)
		}
		first := "security.secret_scanning_push_protection"
		if enabled {
			first = "security.secret_scanning"
		}
		if changes[0].Field() != first {
			t.Fatal(changes)
		}
	}
	for _, body := range []string{`{"security_and_analysis":null}`, `{"security_and_analysis":{"secret_scanning":{"status":42},"secret_scanning_push_protection":{"status":"disabled"}}}`} {
		client := &SecretScanningClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, body), nil }).rest}
		raw, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
		if err == nil {
			t.Fatal("unsupported data accepted")
		}
		if _, err := WithSecurityScanningObservation(uimodel.SecuritySettings{}, raw); err == nil {
			t.Fatal("failure defaulted")
		}
	}
}
