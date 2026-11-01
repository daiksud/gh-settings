package github

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestDependabotFreshToggleReadbackAndAcceptedDisable(t *testing.T) {
	for _, field := range []string{settings.DependabotAlertsField, settings.DependabotUpdatesField} {
		for _, after := range []bool{false, true} {
			for _, mode := range []string{"success", "matching-paused", "pre403", "write403", "write404", "write200", "write-transport", "write-redirect", "readback403", "readback404", "readback-redirect", "readback-malformed", "ignored", "companion-drift", "permission-loss"} {
				t.Run(fmt.Sprint(field, "/", after, "/", mode), func(t *testing.T) {
					a, u := true, false
					target := "/automated-security-fixes"
					if field == settings.DependabotAlertsField {
						a = !after
						target = "/vulnerability-alerts"
					} else {
						u = !after
					}
					if mode == "matching-paused" {
						if field == settings.DependabotAlertsField {
							a = after
						} else {
							u = after
						}
					}
					writes, reads := 0, 0
					var warnings []string
					client := &DependabotClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
						isAlerts := strings.HasSuffix(req.URL.Path, "/vulnerability-alerts")
						if req.URL.Path != "/repos/owner/current/vulnerability-alerts" && req.URL.Path != "/repos/owner/current/automated-security-fixes" {
							t.Fatal("unrelated target", req.URL)
						}
						if req.Method == "GET" {
							reads++
							if mode == "pre403" || writes > 0 && mode == "readback403" {
								return actionsResponse(req, 403, `{}`), nil
							}
							if writes > 0 && mode == "readback-redirect" {
								r := actionsResponse(req, 307, "")
								r.Header.Set("Location", "https://other.invalid/redirect")
								return r, nil
							}
							if writes > 0 && strings.HasSuffix(req.URL.Path, target) && (mode == "readback404" || mode == "permission-loss") {
								return actionsResponse(req, 404, `{"message":"Resource not accessible by integration"}`), nil
							}
							if isAlerts {
								if !a {
									return actionsResponse(req, 404, `{}`), nil
								}
								return actionsResponse(req, 204, ""), nil
							}
							if writes > 0 && mode == "readback-malformed" {
								return actionsResponse(req, 200, `{"enabled":false}`), nil
							}
							return actionsResponse(req, 200, fmt.Sprintf(`{"enabled":%t,"paused":true}`, u)), nil
						}
						want := "DELETE"
						if after {
							want = "PUT"
						}
						if req.Method != want || !strings.HasSuffix(req.URL.Path, target) {
							t.Fatal("undeclared write", req.Method, req.URL)
						}
						if req.Body != nil {
							b, _ := io.ReadAll(req.Body)
							if len(b) > 0 {
								t.Fatal("toggle body", string(b))
							}
						}
						writes++
						if mode == "write-transport" {
							return nil, errors.New("write transport failure")
						}
						for suffix, status := range map[string]int{"write403": 403, "write404": 404, "write200": 200, "write-redirect": 307} {
							if mode == suffix {
								r := actionsResponse(req, status, `{}`)
								r.Header.Set("Location", "https://other.invalid/redirect")
								return r, nil
							}
						}
						if mode != "ignored" && mode != "permission-loss" {
							if isAlerts {
								a = after
							} else {
								u = after
							}
						}
						if mode == "companion-drift" {
							if isAlerts {
								u = !u
							} else {
								a = !a
							}
						}
						return actionsResponse(req, 204, ""), nil
					}).rest, Warning: func(message string) { warnings = append(warnings, message) }}
					change, err := settings.NewSecurityChange(field, !after, after)
					if err != nil {
						t.Fatal(err)
					}
					result, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
					good := mode == "success" || mode == "matching-paused" || !after && (mode == "readback404" || mode == "permission-loss")
					if (err == nil) != good || result.Wrote != (good && mode != "matching-paused") {
						t.Fatal(result, err)
					}
					accepted := good && mode != "matching-paused" && !after && (field == settings.DependabotAlertsField || mode == "readback404" || mode == "permission-loss")
					if result.AcceptedDisable != accepted || accepted && !strings.Contains(strings.Join(warnings, " "), "not independently verified") {
						t.Fatal(result, warnings)
					}
					var pre interface{ ReadFailedBeforeWrite() }
					if errors.As(err, &pre) != (mode == "pre403") {
						t.Fatal("wrong certainty", err)
					}
					wantWrites := 1
					if mode == "pre403" || mode == "matching-paused" {
						wantWrites = 0
					}
					if writes != wantWrites || reads < 1 {
						t.Fatal(writes, reads)
					}
					if mode == "permission-loss" && !after {
						if field == settings.DependabotAlertsField && !a || field == settings.DependabotUpdatesField && !u {
							t.Fatal("fixture must retain enabled to demonstrate accepted uncertainty")
						}
					}
				})
			}
		}
	}
}

func TestDependabotFreshPrerequisiteAndFieldGuards(t *testing.T) {
	for _, field := range []string{settings.DependabotAlertsField, settings.DependabotUpdatesField, "security.secret_scanning"} {
		writes := 0
		client := &DependabotClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" {
				writes++
				t.Fatal("unsafe write")
			}
			if strings.HasSuffix(req.URL.Path, "/vulnerability-alerts") {
				return actionsResponse(req, 404, `{}`), nil
			}
			return actionsResponse(req, 200, `{"enabled":true,"paused":false}`), nil
		}).rest}
		after := field != settings.DependabotAlertsField
		change, _ := settings.NewSecurityChange(field, !after, after)
		result, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, change)
		var pre interface{ ReadFailedBeforeWrite() }
		if err == nil || !errors.As(err, &pre) || result.Wrote || writes != 0 {
			t.Fatal(result, err, writes)
		}
	}
}
