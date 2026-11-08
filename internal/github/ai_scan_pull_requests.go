package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// AIScanPullRequestsClient manages only pull request enablement. Configuration
// members are deliberately absent from its PATCH, including on enablement.
type AIScanPullRequestsClient struct{ rest *api.RESTClient }

func NewAIScanPullRequestsClient(host string) (*AIScanPullRequestsClient, error) {
	client, err := newSecurityRESTClient(host)
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure AI Scan for pull requests API: %w", err)}
	}
	return &AIScanPullRequestsClient{rest: client}, nil
}
func aiScanPullRequestsPath(repo repository.Repository) string {
	return securityRepositoryPath(repo) + "/code-scanning/ai-scan"
}

func (c *AIScanPullRequestsClient) Read(repo repository.Repository) (settings.AIScanPullRequestsObservation, error) {
	if err := readPersonalPublicSecurityTarget(c.rest, repo); err != nil {
		return settings.AIScanPullRequestsObservation{}, err
	}
	var root map[string]json.RawMessage
	if err := c.rest.Get(aiScanPullRequestsPath(repo), &root); err != nil {
		return settings.AIScanPullRequestsObservation{}, fmt.Errorf("read AI Scan for pull requests: %w", err)
	}
	var status *string
	if json.Unmarshal(root["pr_scan"], &status) != nil || status == nil || (*status != "enabled" && *status != "disabled") {
		return settings.AIScanPullRequestsObservation{}, fmt.Errorf("GitHub returned missing/null or unsupported AI Scan for pull requests status")
	}
	return settings.NewAIScanPullRequestsObservation(*status == "enabled"), nil
}

// Update rereads applicability and enablement, writes the one declared member, and
// confirms enablement. Enabling permits AI processing and may consume AI credits;
// readback confirms enablement, never scan completion or consumption.
func (c *AIScanPullRequestsClient) Update(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	before, after := change.Values()
	if change.Field() != "security.ai_scan_pull_requests" {
		return false, &securityPreWriteFailure{fmt.Errorf("change does not belong to AI Scan for pull requests")}
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
	status := "disabled"
	if after {
		status = "enabled"
	}
	body, _ := json.Marshal(map[string]string{"pr_scan": status})
	var response json.RawMessage
	if err := c.rest.Patch(aiScanPullRequestsPath(repo), bytes.NewReader(body), &response); err != nil {
		return false, fmt.Errorf("update AI Scan for pull requests (write effects and AI processing may already have started): %w", err)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify AI Scan for pull requests (write effects and AI processing may already have started): %w", err)
	}
	retained, err := actual.Value()
	if err != nil {
		return false, fmt.Errorf("verify AI Scan for pull requests (write effects may already have applied): %w", err)
	}
	if retained != after {
		return false, fmt.Errorf("AI Scan for pull requests is not confirmed; requested setting may be overridden (write effects and AI processing may already have started)")
	}
	return true, nil
}
