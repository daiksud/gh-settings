package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type SecretScanningClient struct {
	rest                 *api.RESTClient
	NormalizationWarning func(settings.SecurityStatusNormalization)
}
type securityPreWriteFailure struct{ cause error }

func (e *securityPreWriteFailure) Error() string        { return e.cause.Error() }
func (e *securityPreWriteFailure) Unwrap() error        { return e.cause }
func (*securityPreWriteFailure) ReadFailedBeforeWrite() {}
func newSecurityRESTClient(host string) (*api.RESTClient, error) {
	return api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
}
func NewSecretScanningClient(host string) (*SecretScanningClient, error) {
	client, err := newSecurityRESTClient(host)
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure Secret scanning API: %w", err)}
	}
	return &SecretScanningClient{rest: client}, nil
}
func securityRepositoryPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name
}
func (c *SecretScanningClient) Read(repo repository.Repository) (settings.SecretScanningObservation, error) {
	var root map[string]json.RawMessage
	if err := c.rest.Get(securityRepositoryPath(repo), &root); err != nil {
		return settings.SecretScanningObservation{}, fmt.Errorf("read Secret scanning: %w", err)
	}
	var block map[string]json.RawMessage
	if json.Unmarshal(root["security_and_analysis"], &block) != nil || block == nil {
		return settings.SecretScanningObservation{}, fmt.Errorf("GitHub returned hidden/null or invalid security_and_analysis")
	}
	var statuses [2]string
	for i, key := range []string{"secret_scanning", "secret_scanning_push_protection"} {
		var field map[string]json.RawMessage
		var status *string
		if json.Unmarshal(block[key], &field) != nil || field == nil || json.Unmarshal(field["status"], &status) != nil || status == nil {
			return settings.SecretScanningObservation{}, fmt.Errorf("GitHub returned missing/null or unsupported %s status", key)
		}
		statuses[i] = *status
	}
	var values [2]bool
	var records []settings.SecurityStatusNormalization
	for i, control := range []settings.SecurityStatusControl{settings.SecretProtectionControl, settings.PushProtectionControl} {
		prepared, err := prepareSecurityStatusObservation(&statuses[i], control)
		if err != nil {
			return settings.SecretScanningObservation{}, err
		}
		values[i] = *prepared.Value
		if prepared.Normalization != (settings.SecurityStatusNormalization{}) {
			records = append(records, prepared.Normalization)
		}
	}
	return settings.NewSecretScanningObservationWithNormalizations(values[0], values[1], records)
}

// Update writes one declared member after a fresh prerequisite read. It verifies
// the desired value and preserved companion; apparent enforcement success fails.
func (c *SecretScanningClient) Update(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	if change.Field() != "security.secret_scanning" && change.Field() != "security.secret_scanning_push_protection" {
		return false, &securityPreWriteFailure{fmt.Errorf("change does not belong to Secret scanning")}
	}
	_, after := change.Values()
	if err := settings.ValidateSecurityChanges([]settings.SecurityChange{change}); err != nil {
		return false, &securityPreWriteFailure{err}
	}
	current, err := c.Read(repo)
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	c.warnStatusNormalization(current, change.Field())
	expected, err := settings.AfterSecretScanningChange(current, change)
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	matching, err := sameRawSecretScanningStatuses(expected, current)
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	if matching {
		return false, nil
	}
	status := settings.SecurityStatusForValue(after)
	payload := map[string]any{"security_and_analysis": map[string]any{strings.TrimPrefix(change.Field(), "security."): map[string]string{"status": status}}}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, &securityPreWriteFailure{err}
	}
	var ignored json.RawMessage
	if err := c.rest.Patch(securityRepositoryPath(repo), bytes.NewReader(body), &ignored); err != nil {
		return false, fmt.Errorf("update %s (write effects may already have applied): %w", change.Field(), err)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify %s (write effects may already have applied): %w", change.Field(), err)
	}
	c.warnStatusNormalization(actual, change.Field())
	matching, err = sameRawSecretScanningStatuses(expected, actual)
	if err != nil {
		return false, fmt.Errorf("verify %s (write effects may already have applied): %w", change.Field(), err)
	}
	if !matching {
		return false, fmt.Errorf("GitHub did not retain requested %s or its companion (write effects may already have applied)", change.Field())
	}
	return true, nil
}
