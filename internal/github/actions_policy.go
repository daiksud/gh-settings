package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type ActionsPolicyClient struct {
	rest                 *api.RESTClient
	NormalizationWarning func(settings.ObservationNormalization)
}

func newActionsPolicyRESTClient(host string) (*api.RESTClient, error) {
	return api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
}
func NewActionsPolicyClient(host string) (*ActionsPolicyClient, error) {
	client, err := newActionsPolicyRESTClient(host)
	if err != nil {
		return nil, fmt.Errorf("configure Actions policy API: %w", err)
	}
	return &ActionsPolicyClient{rest: client}, nil
}

func actionsPolicyPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/actions/permissions"
}
func actionsBool(fields map[string]json.RawMessage, key string) (bool, error) {
	var v *bool
	if json.Unmarshal(fields[key], &v) != nil || v == nil {
		return false, fmt.Errorf("GitHub returned missing/null or invalid Actions %s", key)
	}
	return *v, nil
}

// ReadActionsPolicy never follows selected_actions_url or reads inactive defaults.
func (c *ActionsPolicyClient) ReadActionsPolicy(repo repository.Repository, selected bool) (settings.ActionsPolicyState, error) {
	var fields map[string]json.RawMessage
	if err := c.rest.Get(actionsPolicyPath(repo), &fields); err != nil {
		return settings.ActionsPolicyState{}, fmt.Errorf("read Actions policy: %w", err)
	}
	var s settings.ActionsPolicyState
	var err error
	if s.Enabled, err = actionsBool(fields, "enabled"); err != nil {
		return settings.ActionsPolicyState{}, err
	}
	if s.SHAPinningRequired, err = actionsBool(fields, "sha_pinning_required"); err != nil {
		return settings.ActionsPolicyState{}, err
	}
	var allowed *string
	if json.Unmarshal(fields["allowed_actions"], &allowed) != nil || allowed == nil {
		return settings.ActionsPolicyState{}, fmt.Errorf("GitHub returned missing/null or invalid Actions allowed_actions")
	}
	s, err = prepareActionsExecutionObservation(&s.Enabled, allowed, &s.SHAPinningRequired)
	if err != nil {
		return settings.ActionsPolicyState{}, err
	}
	if selected && s.AllowedActions == "selected" {
		value, err := c.readSelectedActions(repo)
		if err != nil {
			return settings.ActionsPolicyState{}, err
		}
		s.Selected = &value
	}
	if err := s.Validate(); err != nil {
		return settings.ActionsPolicyState{}, err
	}
	return s, nil
}

func (c *ActionsPolicyClient) readSelectedActions(repo repository.Repository) (settings.SelectedActionsState, error) {
	var fields map[string]json.RawMessage
	if err := c.rest.Get(actionsPolicyPath(repo)+"/selected-actions", &fields); err != nil {
		return settings.SelectedActionsState{}, fmt.Errorf("read selected Actions policy: %w", err)
	}
	var s settings.SelectedActionsState
	var err error
	if s.GitHubOwnedAllowed, err = actionsBool(fields, "github_owned_allowed"); err != nil {
		return s, err
	}
	if s.VerifiedAllowed, err = actionsBool(fields, "verified_allowed"); err != nil {
		return s, err
	}
	if json.Unmarshal(fields["patterns_allowed"], &s.PatternsAllowed) != nil {
		return s, fmt.Errorf("GitHub returned invalid Actions patterns_allowed")
	}
	if err := settings.ValidateActionPatterns(s.PatternsAllowed); err != nil {
		return s, fmt.Errorf("GitHub returned incomplete/invalid selected Actions: %w", err)
	}
	return s, nil
}

func (c *ActionsPolicyClient) putActionsPolicy(path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	response, err := c.rest.Request(http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected Actions policy write status %d", response.StatusCode)
	}
	return nil
}

// ApplyActionsPolicy preflights the whole resource, then writes policy before the
// selected endpoint that requires it. Failures never imply rollback or no writes.
func (c *ActionsPolicyClient) ApplyActionsPolicy(repo repository.Repository, d settings.ActionsPolicyDeclaration) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if d.ExecutionEmpty() {
		return nil
	}
	current, err := c.ReadActionsPolicy(repo, !d.SelectedActions.Empty())
	if err != nil {
		return err
	}
	expected, err := prepareActionsExecutionVerification(current, d)
	if err != nil {
		return err
	}
	c.warnExecutionNormalization(current, d)
	enableAfterSelection := d.Enabled != nil && *d.Enabled && !current.Enabled && !d.SelectedActions.Empty()
	if d.Enabled != nil || d.AllowedActions != nil || d.SHAPinningRequired != nil {
		enabled := current.Enabled
		if d.Enabled != nil && !enableAfterSelection {
			enabled = *d.Enabled
		}
		payload := map[string]any{"enabled": enabled}
		if d.AllowedActions != nil {
			payload["allowed_actions"] = *d.AllowedActions
		}
		if d.SHAPinningRequired != nil {
			payload["sha_pinning_required"] = *d.SHAPinningRequired
		}
		if err := c.putActionsPolicy(actionsPolicyPath(repo), payload); err != nil {
			return fmt.Errorf("write Actions policy (this and earlier writes may already have applied): %w", err)
		}
	}
	if !d.SelectedActions.Empty() {
		selected := expected.selected
		payload := map[string]any{"github_owned_allowed": selected.GitHubOwnedAllowed, "verified_allowed": selected.VerifiedAllowed, "patterns_allowed": selected.PatternsAllowed}
		if err := c.putActionsPolicy(actionsPolicyPath(repo)+"/selected-actions", payload); err != nil {
			return fmt.Errorf("write selected Actions policy (this and earlier policy/resource writes may already have applied): %w", err)
		}
	}
	if enableAfterSelection {
		if err := c.putActionsPolicy(actionsPolicyPath(repo), map[string]any{"enabled": true}); err != nil {
			return fmt.Errorf("enable Actions after selection (this and earlier policy writes may already have applied): %w", err)
		}
	}
	actual, err := c.ReadActionsPolicy(repo, !d.SelectedActions.Empty())
	if err != nil {
		return fmt.Errorf("verify Actions policy (writes may already have applied): %w", err)
	}
	c.warnExecutionNormalization(actual, d)
	retained, err := sameActionsExecutionVerification(expected, actual)
	if err != nil || !retained {
		return fmt.Errorf("GitHub did not retain requested Actions policy (writes may already have applied)")
	}
	return nil
}
