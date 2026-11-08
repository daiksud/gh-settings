package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// CodeQLDefaultSetupClient manages only default setup's state. Configuration
// members are deliberately absent from its PATCH, including on enablement.
type CodeQLDefaultSetupClient struct{ rest *api.RESTClient }

func NewCodeQLDefaultSetupClient(host string) (*CodeQLDefaultSetupClient, error) {
	client, err := newSecurityRESTClient(host)
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure CodeQL default setup API: %w", err)}
	}
	return &CodeQLDefaultSetupClient{rest: client}, nil
}
func codeQLDefaultSetupPath(repo repository.Repository) string {
	return securityRepositoryPath(repo) + "/code-scanning/default-setup"
}

func readPersonalPublicSecurityTarget(rest *api.RESTClient, repo repository.Repository) error {
	var root map[string]json.RawMessage
	if err := rest.Get(securityRepositoryPath(repo), &root); err != nil {
		return fmt.Errorf("read security target applicability: %w", err)
	}
	var visibility *string
	var owner map[string]json.RawMessage
	var ownerType *string
	if json.Unmarshal(root["visibility"], &visibility) != nil || visibility == nil || json.Unmarshal(root["owner"], &owner) != nil || owner == nil || json.Unmarshal(owner["type"], &ownerType) != nil || ownerType == nil {
		return fmt.Errorf("GitHub returned incomplete security target applicability")
	}
	if *visibility != "public" || *ownerType != "User" {
		return fmt.Errorf("this security control requires a personally owned public repository")
	}
	return nil
}

func (c *CodeQLDefaultSetupClient) Read(repo repository.Repository) (settings.CodeQLDefaultSetupObservation, error) {
	if err := readPersonalPublicSecurityTarget(c.rest, repo); err != nil {
		return settings.CodeQLDefaultSetupObservation{}, err
	}
	var root map[string]json.RawMessage
	if err := c.rest.Get(codeQLDefaultSetupPath(repo), &root); err != nil {
		return settings.CodeQLDefaultSetupObservation{}, fmt.Errorf("read CodeQL default setup: %w", err)
	}
	var state *string
	if json.Unmarshal(root["state"], &state) != nil || state == nil || (*state != "configured" && *state != "not-configured") {
		return settings.CodeQLDefaultSetupObservation{}, fmt.Errorf("GitHub returned missing/null or unsupported CodeQL default setup state")
	}
	return settings.NewCodeQLDefaultSetupObservation(*state == "configured"), nil
}

// Update rereads applicability and state, writes the one declared member, and
// confirms configuration state. HTTP 202 may start validation/analysis; readback
// confirms only enablement, never analysis completion or successful execution.
func (c *CodeQLDefaultSetupClient) Update(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	before, after := change.Values()
	if change.Field() != "security.codeql_default_setup" {
		return false, &securityPreWriteFailure{fmt.Errorf("change does not belong to CodeQL default setup")}
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
	state := "not-configured"
	if after {
		state = "configured"
	}
	body, _ := json.Marshal(map[string]string{"state": state})
	var response json.RawMessage
	if err := c.rest.Patch(codeQLDefaultSetupPath(repo), bytes.NewReader(body), &response); err != nil {
		return false, fmt.Errorf("update CodeQL default setup (write effects and analysis may already have started): %w", err)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify CodeQL default setup (write effects and analysis may already have started): %w", err)
	}
	retained, err := actual.Value()
	if err != nil {
		return false, fmt.Errorf("verify CodeQL default setup (write effects may already have applied): %w", err)
	}
	if retained != after {
		return false, fmt.Errorf("CodeQL default setup is not confirmed; asynchronous validation may still be pending (write effects and analysis may already have started)")
	}
	return true, nil
}
