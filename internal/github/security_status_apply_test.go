package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestNormalizedSecurityEqualitySkipsFreshCanonicalMatch(t *testing.T) {
	for _, control := range []settings.SecurityStatusControl{settings.SecretProtectionControl, settings.PushProtectionControl} {
		record := settings.SecurityStatusNormalization{Path: control.Path(), RawValue: "FUTURE", Fallback: false}
		change, err := settings.NewSecurityChangeWithNormalization(control.Field(), false, false, record)
		if err != nil {
			t.Fatal(err)
		}
		reads := 0
		client := &SecretScanningClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" || req.URL.Path != "/repos/owner/current" {
				t.Fatal("unexpected mutation or target", req.Method, req.URL)
			}
			reads++
			return actionsResponse(req, 200, securityResponse(false, false)), nil
		}).rest}
		wrote, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
		if err != nil || wrote || reads != 1 {
			t.Fatal(control, wrote, reads, err)
		}
	}
}

func TestSecurityStatusActivationUsesExactRequestedAndCompanionReadback(t *testing.T) {
	for _, control := range []struct {
		identity               settings.SecurityStatusControl
		key, companion, status string
	}{
		{settings.SecretProtectionControl, "secret_scanning", "secret_scanning_push_protection", "disabled"},
		{settings.PushProtectionControl, "secret_scanning_push_protection", "secret_scanning", "enabled"},
	} {
		for _, raw := range []string{"FUTURE", "", " \t"} {
			for _, desired := range []bool{false, true} {
				for _, mode := range []string{"applied", "retained unknown", "companion changed"} {
					t.Run(string(control.identity)+raw+mode+map[bool]string{false: " false", true: " true"}[desired], func(t *testing.T) {
						statuses := map[string]string{control.key: raw, control.companion: control.status}
						reads, writes := 0, 0
						client := &SecretScanningClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
							if req.URL.Path != "/repos/owner/current" {
								t.Fatal(req.URL)
							}
							if req.Method == "GET" {
								reads++
								block := map[string]any{}
								for key, status := range statuses {
									block[key] = map[string]string{"status": status}
								}
								body, _ := json.Marshal(map[string]any{"security_and_analysis": block})
								return actionsResponse(req, 200, string(body)), nil
							}
							if req.Method != "PATCH" {
								t.Fatal(req.Method)
							}
							writes++
							var payload map[string]any
							if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
								t.Fatal(err)
							}
							status := map[bool]string{false: "disabled", true: "enabled"}[desired]
							want := map[string]any{"security_and_analysis": map[string]any{control.key: map[string]any{"status": status}}}
							if !reflect.DeepEqual(payload, want) {
								t.Fatal(payload, want)
							}
							if mode != "retained unknown" {
								statuses[control.key] = status
							}
							if mode == "companion changed" {
								statuses[control.companion] = map[string]string{"enabled": "disabled", "disabled": "enabled"}[control.status]
							}
							return actionsResponse(req, 200, `{}`), nil
						}).rest}
						var warnings []settings.SecurityStatusNormalization
						client.NormalizationWarning = func(record settings.SecurityStatusNormalization) { warnings = append(warnings, record) }
						repo := repository.Repository{Owner: "owner", Name: "current"}
						observed, err := client.Read(repo)
						if err != nil || len(warnings) != 0 {
							t.Fatal("pure read failed or warned", observed, warnings, err)
						}
						record := settings.SecurityStatusNormalization{Path: control.identity.Path(), RawValue: raw, Fallback: false}
						if !reflect.DeepEqual(observed.Normalizations(), []settings.SecurityStatusNormalization{record}) {
							t.Fatal(observed.Normalizations())
						}
						change, err := settings.NewSecurityChangeWithNormalization(control.identity.Field(), false, desired, record)
						if err != nil {
							t.Fatal(err)
						}
						wrote, err := client.Update(repo, change)
						var pre interface{ ReadFailedBeforeWrite() }
						if wrote != (mode == "applied") || (err == nil) != (mode == "applied") || errors.As(err, &pre) || err != nil && !strings.Contains(err.Error(), "write effects may already have applied") {
							t.Fatal(wrote, err)
						}
						wantWarnings := 1
						if mode == "retained unknown" {
							wantWarnings = 2
						}
						if reads != 3 || writes != 1 || len(warnings) != wantWarnings {
							t.Fatal(reads, writes, warnings)
						}
						for _, warning := range warnings {
							if warning != record {
								t.Fatal(warning, record)
							}
						}
					})
				}
			}
		}
	}
}

func TestSecurityUnknownCompanionIsPreservedAndNeverProvesPrerequisites(t *testing.T) {
	for _, test := range []struct {
		field, key, companion string
		before, after         bool
		prerequisite          bool
	}{
		{"security.secret_scanning", "secret_scanning", "secret_scanning_push_protection", false, true, false},
		{"security.secret_scanning_push_protection", "secret_scanning_push_protection", "secret_scanning", true, false, false},
		{"security.secret_scanning", "secret_scanning", "secret_scanning_push_protection", true, false, true},
		{"security.secret_scanning_push_protection", "secret_scanning_push_protection", "secret_scanning", false, true, true},
	} {
		for _, drift := range []bool{false, true} {
			statuses := map[string]string{test.key: map[bool]string{false: "disabled", true: "enabled"}[test.before], test.companion: "FUTURE_COMPANION"}
			writes := 0
			client := &SecretScanningClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/repos/owner/current" {
					t.Fatal(req.URL)
				}
				if req.Method == "GET" {
					block := map[string]any{}
					for key, status := range statuses {
						block[key] = map[string]string{"status": status}
					}
					body, _ := json.Marshal(map[string]any{"security_and_analysis": block})
					return actionsResponse(req, 200, string(body)), nil
				}
				if req.Method != "PATCH" {
					t.Fatal(req.Method)
				}
				writes++
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				status := map[bool]string{false: "disabled", true: "enabled"}[test.after]
				if !reflect.DeepEqual(payload, map[string]any{"security_and_analysis": map[string]any{test.key: map[string]any{"status": status}}}) {
					t.Fatal(payload)
				}
				statuses[test.key] = status
				if drift {
					statuses[test.companion] = "OTHER_UNKNOWN"
				}
				return actionsResponse(req, 200, `{}`), nil
			}).rest}
			warnings := 0
			client.NormalizationWarning = func(settings.SecurityStatusNormalization) { warnings++ }
			change, err := settings.NewSecurityChange(test.field, test.before, test.after)
			if err != nil {
				t.Fatal(err)
			}
			wrote, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
			var pre interface{ ReadFailedBeforeWrite() }
			if test.prerequisite {
				if wrote || err == nil || writes != 0 || !errors.As(err, &pre) {
					t.Fatal("unknown proved prerequisite", test, wrote, writes, err)
				}
			} else if wrote != !drift || (err != nil) != drift || writes != 1 || errors.As(err, &pre) {
				t.Fatal("companion verification", test, wrote, writes, err)
			}
			if warnings != 0 {
				t.Fatal("unmanaged companion leaked", warnings)
			}
		}
	}
}

func TestSecurityIncompleteSecondMemberDiscardsNormalization(t *testing.T) {
	for _, raw := range []string{`null`, `true`, `42`, `{}`} {
		body := `{"security_and_analysis":{"secret_scanning":{"status":"FUTURE"},"secret_scanning_push_protection":{"status":` + raw + `}}}`
		client := &SecretScanningClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			return actionsResponse(req, 200, body), nil
		}).rest}
		warnings := 0
		client.NormalizationWarning = func(settings.SecurityStatusNormalization) { warnings++ }
		observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
		if err == nil || observed != (settings.SecretScanningObservation{}) || warnings != 0 {
			t.Fatal(observed, warnings, err)
		}
	}
}

func TestSecurityInconsistentProjectionIsAcquiredButNotDeclarable(t *testing.T) {
	client := &SecretScanningClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" || req.URL.Path != "/repos/owner/current" {
			t.Fatal(req.Method, req.URL)
		}
		return actionsResponse(req, 200, `{"security_and_analysis":{"secret_scanning":{"status":"FUTURE"},"secret_scanning_push_protection":{"status":"enabled"}}}`), nil
	}).rest}
	warnings := 0
	client.NormalizationWarning = func(settings.SecurityStatusNormalization) { warnings++ }
	observed, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
	if err != nil || warnings != 0 {
		t.Fatal(observed, warnings, err)
	}
	scanning, push, err := observed.Values()
	if err != nil || scanning || !push {
		t.Fatal(scanning, push, err)
	}
	if !reflect.DeepEqual(observed.Normalizations(), []settings.SecurityStatusNormalization{{Path: "repository.settings.advanced_security.secret_protection.enabled", RawValue: "FUTURE", Fallback: false}}) {
		t.Fatal(observed.Normalizations())
	}
	if _, err := observed.Declaration(); err == nil {
		t.Fatal("inconsistent projected declaration accepted")
	}
}
