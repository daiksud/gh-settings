package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type DependabotClient struct {
	rest    *api.RESTClient
	Warning func(string)
}

func (c *DependabotClient) warn(message string) {
	if c.Warning != nil {
		c.Warning(message)
	}
}

func NewDependabotClient(host string) (*DependabotClient, error) {
	client, err := newSecurityRESTClient(host)
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure Dependabot API: %w", err)}
	}
	return &DependabotClient{rest: client}, nil
}

func (c *DependabotClient) ReadAlerts(repo repository.Repository) (settings.DependabotAlertsObservation, error) {
	response, err := c.rest.Request(http.MethodGet, securityRepositoryPath(repo)+"/vulnerability-alerts", nil)
	if err != nil {
		if dependabotDisabled404(err) {
			c.warn("Dependabot alerts GET 404 is interpreted as disabled; insufficient permission may be misclassified.")
			return settings.NewDependabotAlerts404Observation(), nil
		}
		return settings.DependabotAlertsObservation{}, fmt.Errorf("read Dependabot alerts: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return settings.DependabotAlertsObservation{}, fmt.Errorf("unexpected Dependabot alerts read status %d", response.StatusCode)
	}
	return settings.NewEnabledDependabotAlertsObservation(), nil
}

func (c *DependabotClient) ReadSecurityUpdates(repo repository.Repository) (settings.DependabotSecurityUpdatesObservation, error) {
	response, err := c.rest.Request(http.MethodGet, securityRepositoryPath(repo)+"/automated-security-fixes", nil)
	if err != nil {
		if dependabotDisabled404(err) {
			c.warn("Dependabot security updates GET 404 is interpreted as disabled; insufficient permission may be misclassified and pause state is unknown.")
			return settings.NewDependabotSecurityUpdates404Observation(), nil
		}
		return settings.DependabotSecurityUpdatesObservation{}, fmt.Errorf("read Dependabot security updates: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return settings.DependabotSecurityUpdatesObservation{}, fmt.Errorf("unexpected Dependabot security updates read status %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return settings.DependabotSecurityUpdatesObservation{}, fmt.Errorf("read Dependabot security updates response: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return settings.DependabotSecurityUpdatesObservation{}, fmt.Errorf("decode Dependabot security updates: %w", err)
	}
	var enabled, paused *bool
	if json.Unmarshal(fields["enabled"], &enabled) != nil || enabled == nil ||
		json.Unmarshal(fields["paused"], &paused) != nil || paused == nil {
		return settings.DependabotSecurityUpdatesObservation{}, fmt.Errorf("GitHub returned missing/null or invalid Dependabot security updates enabled/paused")
	}
	if *paused {
		c.warn("Dependabot security updates are paused; enabled state does not mean updates are running. No resume is requested.")
	}
	return settings.NewDependabotSecurityUpdatesObservation(*enabled, *paused), nil
}

func (c *DependabotClient) Read(repo repository.Repository) (settings.DependabotObservation, error) {
	a, err := c.ReadAlerts(repo)
	if err != nil {
		return settings.DependabotObservation{}, err
	}
	u, err := c.ReadSecurityUpdates(repo)
	if err != nil {
		return settings.DependabotObservation{}, err
	}
	return settings.DependabotObservation{Alerts: a, Updates: u}, nil
}

// Update preflights a fresh complete family and sends one body-free toggle.
// DELETE 204 followed by target GET 404 is explicitly accepted with uncertainty.
// PUT never accepts 404 as evidence that enabled=true was retained.
func (c *DependabotClient) Update(repo repository.Repository, change settings.SecurityChange) (settings.DependabotWriteResult, error) {
	failed := func(err error) (settings.DependabotWriteResult, error) {
		return settings.DependabotWriteResult{}, &securityPreWriteFailure{err}
	}
	if change.Field() != settings.DependabotAlertsField && change.Field() != settings.DependabotUpdatesField {
		return failed(fmt.Errorf("change does not belong to Dependabot"))
	}
	if err := settings.ValidateSecurityChanges([]settings.SecurityChange{change}); err != nil {
		return failed(err)
	}
	_, after := change.Values()
	current, err := c.Read(repo)
	if err != nil {
		return failed(err)
	}
	a, u, err := current.Values()
	if err != nil {
		return failed(err)
	}
	d := settings.SecurityDeclaration{}
	path := "/vulnerability-alerts"
	before := a
	if change.Field() == settings.DependabotAlertsField {
		d.DependabotAlerts = &after
	} else {
		d.DependabotSecurityUpdates = &after
		path = "/automated-security-fixes"
		before = u
	}
	if _, _, err := settings.DependabotTransitions(current, d); err != nil {
		return failed(err)
	}
	if before == after {
		return settings.DependabotWriteResult{}, nil
	}
	method := http.MethodDelete
	if after {
		method = http.MethodPut
	}
	response, err := c.rest.Request(method, securityRepositoryPath(repo)+path, nil)
	if err != nil {
		return settings.DependabotWriteResult{}, fmt.Errorf("update Dependabot (write effects may already have applied): %w", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return settings.DependabotWriteResult{}, fmt.Errorf("unexpected Dependabot write status %d (write effects may already have applied)", response.StatusCode)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return settings.DependabotWriteResult{}, fmt.Errorf("read back Dependabot (write effects may already have applied): %w", err)
	}
	aa, uu, err := actual.Values()
	if err != nil {
		return settings.DependabotWriteResult{}, fmt.Errorf("read back Dependabot (write effects may already have applied): %w", err)
	}
	accepted := false
	if change.Field() == settings.DependabotAlertsField {
		if aa != after || uu != u {
			return settings.DependabotWriteResult{}, fmt.Errorf("Dependabot alerts or preserved security updates differ after write (write effects may already have applied)")
		}
		accepted = !after && actual.Alerts.DisabledBy404()
		if !accepted {
			err = actual.Alerts.VerifyEnabled(after)
		}
	} else {
		if uu != after || aa != a {
			return settings.DependabotWriteResult{}, fmt.Errorf("Dependabot security updates or preserved alerts differ after write (write effects may already have applied)")
		}
		accepted = !after && actual.Updates.DisabledBy404()
		if !accepted {
			err = actual.Updates.VerifyEnabled(after)
		}
	}
	if err != nil {
		return settings.DependabotWriteResult{}, fmt.Errorf("verify Dependabot (write effects may already have applied): %w", err)
	}
	if accepted {
		c.warn("Dependabot disable accepted after DELETE 204 and GET 404; retained disabled state is not independently verified and permission loss or enforcement cannot be excluded.")
	}
	return settings.DependabotWriteResult{Wrote: true, AcceptedDisable: accepted}, nil
}

func dependabotDisabled404(err error) bool {
	var status *api.HTTPError
	// Explicit product decision, not an authorization test. Permission-concealing
	// 404 responses are intentionally interpreted as disabled as well.
	return errors.As(err, &status) && status.StatusCode == http.StatusNotFound
}
