package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// AIScanRepositoriesClient changes configuration, never requests an AI scan.
type AIScanRepositoriesClient struct{ rest *api.RESTClient }

func NewAIScanRepositoriesClient(host string) (*AIScanRepositoriesClient, error) {
	rest, err := newSecurityRESTClient(host)
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure AI Scan for repositories client: %w", err)}
	}
	return &AIScanRepositoriesClient{rest: rest}, nil
}

type aiScanRepositoryState struct {
	id, ownerID           int64
	enabled, pullRequests bool
}

func (c *AIScanRepositoriesClient) get(path string, out any) error {
	response, err := c.rest.Request(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("AI Scan observation unavailable")
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	var extra any
	if response.StatusCode != http.StatusOK || decoder.Decode(out) != nil || decoder.Decode(&extra) != io.EOF {
		return fmt.Errorf("AI Scan observation incomplete or unavailable")
	}
	return nil
}

func decodeAIScanRepositories(raw map[string]json.RawMessage) (bool, bool, error) {
	values := map[string]bool{}
	for _, key := range []string{"repository_scan", "pr_scan"} {
		var status *string
		if json.Unmarshal(raw[key], &status) != nil || status == nil || (*status != "enabled" && *status != "disabled") {
			return false, false, fmt.Errorf("AI Scan response missing or unsupported %s", key)
		}
		values[key] = *status == "enabled"
	}
	return values["repository_scan"], values["pr_scan"], nil
}

func (c *AIScanRepositoriesClient) read(repo repository.Repository, companion bool) (aiScanRepositoryState, error) {
	var target struct {
		ID         *int64  `json:"id"`
		FullName   *string `json:"full_name"`
		Visibility *string `json:"visibility"`
		Owner      *struct {
			ID   *int64  `json:"id"`
			Type *string `json:"type"`
		} `json:"owner"`
	}
	if err := c.get(securityRepositoryPath(repo), &target); err != nil {
		return aiScanRepositoryState{}, err
	}
	if target.ID == nil || *target.ID <= 0 || target.FullName == nil || !strings.EqualFold(*target.FullName, repo.Owner+"/"+repo.Name) || target.Visibility == nil || *target.Visibility != "public" || target.Owner == nil || target.Owner.ID == nil || *target.Owner.ID <= 0 || target.Owner.Type == nil || *target.Owner.Type != "User" {
		return aiScanRepositoryState{}, fmt.Errorf("AI Scan for repositories requires complete personal-public repository identity")
	}
	var raw map[string]json.RawMessage
	if err := c.get(aiScanPullRequestsPath(repo), &raw); err != nil {
		return aiScanRepositoryState{}, err
	}
	var enabled, pr bool
	if companion {
		var err error
		enabled, pr, err = decodeAIScanRepositories(raw)
		if err != nil {
			return aiScanRepositoryState{}, err
		}
	} else {
		var status *string
		if json.Unmarshal(raw["repository_scan"], &status) != nil || status == nil || (*status != "enabled" && *status != "disabled") {
			return aiScanRepositoryState{}, fmt.Errorf("AI Scan response missing or unsupported repository_scan")
		}
		enabled = *status == "enabled"
	}
	return aiScanRepositoryState{id: *target.ID, ownerID: *target.Owner.ID, enabled: enabled, pullRequests: pr}, nil
}

func (c *AIScanRepositoriesClient) Read(repo repository.Repository) (settings.AIScanRepositoriesObservation, error) {
	observed, err := c.read(repo, false)
	if err != nil {
		return settings.AIScanRepositoriesObservation{}, err
	}
	return settings.NewAIScanRepositoriesObservation(observed.enabled), nil
}

func (c *AIScanRepositoriesClient) Update(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	before, after := change.Values()
	pre := func(err error) (bool, error) { return false, &securityPreWriteFailure{err} }
	if change.Field() != "security.ai_scan_repositories" {
		return pre(fmt.Errorf("change does not belong to AI Scan for repositories"))
	}
	if _, err := settings.NewSecurityChange(change.Field(), before, after); err != nil {
		return pre(err)
	}
	current, err := c.read(repo, true)
	if err != nil {
		return pre(err)
	}
	if current.enabled == after {
		return false, nil
	}
	status := "disabled"
	if after {
		status = "enabled"
	}
	body, _ := json.Marshal(map[string]string{"repository_scan": status})
	response, err := c.rest.Request(http.MethodPatch, aiScanPullRequestsPath(repo), bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("update AI Scan for repositories failed; configuration may have changed")
	}
	decoder := json.NewDecoder(response.Body)
	var raw map[string]json.RawMessage
	var extra any
	valid := response.StatusCode == http.StatusOK && decoder.Decode(&raw) == nil && decoder.Decode(&extra) == io.EOF
	response.Body.Close()
	enabled, pr, ackErr := decodeAIScanRepositories(raw)
	if !valid || ackErr != nil || enabled != after || pr != current.pullRequests {
		return false, fmt.Errorf("AI Scan acknowledgment did not confirm requested state and preserved PR companion; configuration may have changed")
	}
	actual, err := c.read(repo, true)
	if err != nil {
		return false, fmt.Errorf("read back AI Scan for repositories failed; configuration may have changed: %w", err)
	}
	if actual.id != current.id || actual.ownerID != current.ownerID || actual.enabled != after || actual.pullRequests != current.pullRequests {
		return false, fmt.Errorf("AI Scan readback did not confirm original identity, requested setting and preserved PR companion; configuration may have changed")
	}
	return true, nil
}
