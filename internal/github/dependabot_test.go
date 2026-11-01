package github

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func dependabotFixture(t *testing.T, path string, status int, body string) *DependabotClient {
	t.Helper()
	calls := 0
	t.Cleanup(func() {
		if calls != 1 {
			t.Errorf("wanted one selected read, got %d", calls)
		}
	})
	return &DependabotClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.URL.Host != "api.github.com" || req.URL.Path != "/repos/owner/current/"+path || req.Body != nil {
			t.Fatal("unexpected request, mutation or credential discovery", req.Method, req.URL)
		}
		response := actionsResponse(req, status, body)
		response.Header.Set("Location", "https://other.invalid/changed-target")
		return response, nil
	}).rest}
}

func TestDependabotAlertsReadStatusPolicy(t *testing.T) {
	for _, test := range []struct {
		status  int
		body    string
		valid   bool
		enabled bool
	}{
		{204, "", true, true},
		{404, `{"message":"Not Found"}`, true, false},
		{404, `{"message":"Resource not accessible by integration"}`, true, false},
		{404, "not JSON", true, false},
		{200, `{"enabled":false}`, false, false},
		{202, "", false, false},
		{205, "", false, false},
		{301, "", false, false},
		{307, "", false, false},
		{401, `{}`, false, false},
		{403, `{}`, false, false},
		{429, `{}`, false, false},
		{500, `{}`, false, false},
	} {
		t.Run(fmt.Sprint(test.status, "/", test.body), func(t *testing.T) {
			client := dependabotFixture(t, "vulnerability-alerts", test.status, test.body)
			observed, err := client.ReadAlerts(repository.Repository{Owner: "owner", Name: "current"})
			if (err == nil) != test.valid {
				t.Fatal(observed, err)
			}
			if !test.valid {
				if observed != (settings.DependabotAlertsObservation{}) {
					t.Fatal("failed read acquired state", observed)
				}
				return
			}
			enabled, err := observed.Value()
			if err != nil || enabled != test.enabled || observed.DisabledBy404() != (test.status == 404) {
				t.Fatal(observed, err)
			}
			if (observed.VerifyEnabled(test.enabled) == nil) != (test.status == 204) {
				t.Fatal("404 is a reading decision, not update verification", observed)
			}
		})
	}
}

func TestDependabotUpdatesCompleteBooleanPairs(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, paused := range []bool{false, true} {
			t.Run(fmt.Sprint(enabled, "/", paused), func(t *testing.T) {
				body := fmt.Sprintf(`{"enabled":%t,"paused":%t,"unowned":"ignored"}`, enabled, paused)
				client := dependabotFixture(t, "automated-security-fixes", 200, body)
				observed, err := client.ReadSecurityUpdates(repository.Repository{Owner: "owner", Name: "current"})
				gotEnabled, gotPaused, known, valueErr := observed.Values()
				if err != nil || valueErr != nil || gotEnabled != enabled || gotPaused != paused || !known || observed.DisabledBy404() {
					t.Fatal(observed, err, valueErr)
				}
				if observed.VerifyEnabled(enabled) != nil || observed.VerifyEnabled(!enabled) == nil {
					t.Fatal("pause is independent of requested enablement", observed)
				}
			})
		}
	}
}

func TestDependabotUpdates404AndMalformedReads(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
	}{
		{404, `{"message":"Not Found"}`},
		{404, `{"message":"Resource not accessible by integration"}`},
		{404, "not JSON"},
		{200, `{}`},
		{200, `null`},
		{200, `[]`},
		{200, `{"enabled":false}`},
		{200, `{"paused":false}`},
		{200, `{"enabled":null,"paused":false}`},
		{200, `{"enabled":false,"paused":null}`},
		{200, `{"enabled":"false","paused":false}`},
		{200, `{"enabled":false,"paused":0}`},
		{200, `{"Enabled":false,"paused":false}`},
		{200, `{"enabled":false,"Paused":false}`},
		{200, `{"enabled":false,"paused":false}{}`},
		{200, `{"enabled":false,"paused":`},
		{204, ""},
		{205, ""},
		{302, ""},
		{308, ""},
		{401, `{}`},
		{403, `{}`},
		{429, `{}`},
		{500, `{}`},
	} {
		t.Run(fmt.Sprint(test.status, "/", test.body), func(t *testing.T) {
			client := dependabotFixture(t, "automated-security-fixes", test.status, test.body)
			observed, err := client.ReadSecurityUpdates(repository.Repository{Owner: "owner", Name: "current"})
			if test.status != 404 {
				if err == nil || observed != (settings.DependabotSecurityUpdatesObservation{}) {
					t.Fatal("failed read acquired state", observed, err)
				}
				return
			}
			enabled, paused, known, valueErr := observed.Values()
			if err != nil || valueErr != nil || enabled || paused || known || !observed.DisabledBy404() {
				t.Fatal("404 must acquire disabled but no pause value", observed, err, valueErr)
			}
			if observed.VerifyEnabled(false) == nil {
				t.Fatal("a 404 after DELETE must not confirm success")
			}
		})
	}
}

func TestDependabotReadFailuresRetainCause(t *testing.T) {
	failure := errors.New("fixture transport failure")
	client := &DependabotClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		return nil, failure
	}).rest}
	repo := repository.Repository{Owner: "owner", Name: "current"}
	alerts, err := client.ReadAlerts(repo)
	if !errors.Is(err, failure) || alerts != (settings.DependabotAlertsObservation{}) {
		t.Fatal(alerts, err)
	}
	updates, err := client.ReadSecurityUpdates(repo)
	if !errors.Is(err, failure) || updates != (settings.DependabotSecurityUpdatesObservation{}) {
		t.Fatal(updates, err)
	}
	client.rest = actionsClient(t, func(req *http.Request) (*http.Response, error) {
		return actionsResponse(req, 403, `{"message":"denied"}`), nil
	}).rest
	_, err = client.ReadSecurityUpdates(repo)
	var denied *api.HTTPError
	if !errors.As(err, &denied) || denied.StatusCode != 403 {
		t.Fatal("lost permission error", err)
	}
}

type dependabotBrokenBody struct{ closed bool }

func (*dependabotBrokenBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (b *dependabotBrokenBody) Close() error           { b.closed = true; return nil }

func TestDependabotUpdatesBodyFailureDoesNotAcquireState(t *testing.T) {
	body := &dependabotBrokenBody{}
	client := &DependabotClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		response := actionsResponse(req, 200, "")
		response.Body = body
		return response, nil
	}).rest}
	observed, err := client.ReadSecurityUpdates(repository.Repository{Owner: "owner", Name: "current"})
	if err == nil || !strings.Contains(err.Error(), "unexpected EOF") || !body.closed || observed != (settings.DependabotSecurityUpdatesObservation{}) {
		t.Fatal(observed, err, body.closed)
	}
}
