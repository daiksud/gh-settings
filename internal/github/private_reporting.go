package github

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type PrivateReportingClient struct{ rest *api.RESTClient }

func NewPrivateReportingClient(host string) (*PrivateReportingClient, error) {
	client, err := newSecurityRESTClient(host)
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure private vulnerability reporting API: %w", err)}
	}
	return &PrivateReportingClient{rest: client}, nil
}
func privateReportingPath(repo repository.Repository) string {
	return securityRepositoryPath(repo) + "/private-vulnerability-reporting"
}
func (c *PrivateReportingClient) Read(repo repository.Repository) (settings.PrivateReportingObservation, error) {
	var metadata map[string]json.RawMessage
	if err := c.rest.Get(securityRepositoryPath(repo), &metadata); err != nil {
		return settings.PrivateReportingObservation{}, fmt.Errorf("read visibility for private vulnerability reporting: %w", err)
	}
	var visibility *string
	if json.Unmarshal(metadata["visibility"], &visibility) != nil || visibility == nil || (*visibility != "public" && *visibility != "private" && *visibility != "internal") {
		return settings.PrivateReportingObservation{}, fmt.Errorf("GitHub returned missing/null or unsupported repository visibility")
	}
	if *visibility != "public" {
		return settings.PrivateReportingObservation{}, fmt.Errorf("private vulnerability reporting is inapplicable to %s repositories", *visibility)
	}
	var fields map[string]json.RawMessage
	if err := c.rest.Get(privateReportingPath(repo), &fields); err != nil {
		return settings.PrivateReportingObservation{}, fmt.Errorf("read private vulnerability reporting: %w", err)
	}
	var enabled *bool
	if json.Unmarshal(fields["enabled"], &enabled) != nil || enabled == nil {
		return settings.PrivateReportingObservation{}, fmt.Errorf("GitHub returned missing/null or invalid private vulnerability reporting enabled")
	}
	return settings.NewPrivateReportingObservation(*enabled), nil
}

// Update checks fresh public applicability and state before its single toggle,
// then verifies the complete observation without retrying or rolling back.
func (c *PrivateReportingClient) Update(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	before, after := change.Values()
	if change.Field() != "security.private_vulnerability_reporting" {
		return false, &securityPreWriteFailure{fmt.Errorf("change does not belong to private vulnerability reporting")}
	}
	if _, err := settings.NewSecurityChange(change.Field(), before, after); err != nil {
		return false, &securityPreWriteFailure{err}
	}
	current, err := c.Read(repo)
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	enabled, err := current.Value()
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	if enabled == after {
		return false, nil
	}
	if after {
		err = c.rest.Put(privateReportingPath(repo), nil, nil)
	} else {
		err = c.rest.Delete(privateReportingPath(repo), nil)
	}
	if err != nil {
		return false, fmt.Errorf("update private vulnerability reporting (write effects may already have applied): %w", err)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify private vulnerability reporting (write effects may already have applied): %w", err)
	}
	retained, err := actual.Value()
	if err != nil {
		return false, fmt.Errorf("verify private vulnerability reporting (write effects may already have applied): %w", err)
	}
	if retained != after {
		return false, fmt.Errorf("GitHub did not retain requested private vulnerability reporting (write effects may already have applied)")
	}
	return true, nil
}
