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

type ImmutableReleasesClient struct{ rest *api.RESTClient }

func NewImmutableReleasesClient(host string) (*ImmutableReleasesClient, error) {
	client, err := newSecurityRESTClient(host)
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure immutable releases API: %w", err)}
	}
	return &ImmutableReleasesClient{rest: client}, nil
}

func immutableReleasesPath(repo repository.Repository) string {
	return securityRepositoryPath(repo) + "/immutable-releases"
}

func (c *ImmutableReleasesClient) Read(repo repository.Repository) (settings.ImmutableReleasesObservation, error) {
	response, err := c.rest.Request(http.MethodGet, immutableReleasesPath(repo), nil)
	if err != nil {
		var status *api.HTTPError
		if errors.As(err, &status) && status.StatusCode == http.StatusNotFound {
			// Explicit product decision: no extra authorization discriminator.
			// An authorization-related 404 can therefore be misread as disabled.
			return settings.NewDisabledImmutableReleasesObservation(), nil
		}
		return settings.ImmutableReleasesObservation{}, fmt.Errorf("read immutable releases: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return settings.ImmutableReleasesObservation{}, fmt.Errorf("unexpected immutable releases read status %d", response.StatusCode)
	}
	var fields map[string]json.RawMessage
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return settings.ImmutableReleasesObservation{}, fmt.Errorf("read immutable releases response: %w", err)
	}
	if err := json.Unmarshal(body, &fields); err != nil {
		return settings.ImmutableReleasesObservation{}, fmt.Errorf("decode immutable releases: %w", err)
	}
	var enabled, enforced *bool
	if json.Unmarshal(fields["enabled"], &enabled) != nil || enabled == nil ||
		json.Unmarshal(fields["enforced_by_owner"], &enforced) != nil || enforced == nil {
		return settings.ImmutableReleasesObservation{}, fmt.Errorf("GitHub returned missing/null or invalid immutable releases enabled/enforced_by_owner")
	}
	observed := settings.NewImmutableReleasesObservation(*enabled, *enforced)
	if _, _, _, err := observed.Values(); err != nil {
		return settings.ImmutableReleasesObservation{}, err
	}
	return observed, nil
}

// Update uses only the policy endpoint, with fresh preflight and readback.
// Neither owner enforcement nor individual releases/tags/assets are mutated.
func (c *ImmutableReleasesClient) Update(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	before, after := change.Values()
	if change.Field() != "security.immutable_releases" {
		return false, &securityPreWriteFailure{fmt.Errorf("change does not belong to immutable releases")}
	}
	if _, err := settings.NewSecurityChange(change.Field(), before, after); err != nil {
		return false, &securityPreWriteFailure{err}
	}
	current, err := c.Read(repo)
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	desired := settings.SecurityDeclaration{ImmutableReleases: &after}
	_, changes, err := settings.CompareImmutableReleases(current, desired)
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	if len(changes) == 0 {
		return false, nil
	}
	method := http.MethodDelete
	if after {
		method = http.MethodPut
	}
	response, err := c.rest.Request(method, immutableReleasesPath(repo), nil)
	if err != nil {
		return false, fmt.Errorf("update immutable releases (write effects may already have applied): %w", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return false, fmt.Errorf("unexpected immutable releases write status %d (write effects may already have applied)", response.StatusCode)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify immutable releases (write effects may already have applied): %w", err)
	}
	enabled, _, _, err := actual.Values()
	if err != nil {
		return false, fmt.Errorf("verify immutable releases (write effects may already have applied): %w", err)
	}
	if enabled != after {
		return false, fmt.Errorf("GitHub did not retain requested immutable releases (write effects may already have applied)")
	}
	return true, nil
}
