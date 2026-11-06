package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type ActionsAccessClient struct{ rest *api.RESTClient }

func NewActionsAccessClient(host string) (*ActionsAccessClient, error) {
	c, err := newActionsPolicyRESTClient(host)
	if err != nil {
		return nil, &actionsGeneralPreWriteFailure{err}
	}
	return &ActionsAccessClient{rest: c}, nil
}
func (c *ActionsAccessClient) Read(repo repository.Repository) (settings.ActionsAccessObservation, error) {
	visibility, fields, err := readActionsVisibility(c.rest, repo)
	if err != nil {
		return settings.ActionsAccessObservation{}, err
	}
	if visibility != "private" {
		return settings.ActionsAccessObservation{}, fmt.Errorf("Actions access requires private repository visibility; observed %s", visibility)
	}
	var owner map[string]json.RawMessage
	var ownerType *string
	if json.Unmarshal(fields["owner"], &owner) != nil || json.Unmarshal(owner["type"], &ownerType) != nil || ownerType == nil || (*ownerType != "User" && *ownerType != "Organization") {
		return settings.ActionsAccessObservation{}, fmt.Errorf("missing/null or invalid repository owner type for Actions access")
	}
	var access map[string]json.RawMessage
	if err := c.rest.Get(actionsPolicyPath(repo)+"/access", &access); err != nil {
		return settings.ActionsAccessObservation{}, fmt.Errorf("read Actions access: %w", err)
	}
	var level *string
	if json.Unmarshal(access["access_level"], &level) != nil || level == nil {
		return settings.ActionsAccessObservation{}, fmt.Errorf("missing/null or invalid Actions access_level")
	}
	return settings.NewActionsAccessObservation(*level, visibility, *ownerType)
}
func (c *ActionsAccessClient) Update(repo repository.Repository, level string) (bool, error) {
	if err := settings.ValidateActionsAccessLevel(level); err != nil {
		return false, &actionsGeneralPreWriteFailure{err}
	}
	current, err := c.Read(repo)
	if err != nil {
		return false, &actionsGeneralPreWriteFailure{err}
	}
	if err := settings.PrepareActionsAccess(current, level); err != nil {
		return false, &actionsGeneralPreWriteFailure{err}
	}
	before, _, _, _ := current.Values()
	if before == level {
		return false, nil
	}
	payload, _ := json.Marshal(map[string]string{"access_level": level})
	if err := c.rest.Put(actionsPolicyPath(repo)+"/access", bytes.NewReader(payload), nil); err != nil {
		return false, fmt.Errorf("write Actions access (write effects may already have applied): %w", err)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify Actions access (write effects may already have applied): %w", err)
	}
	after, _, _, err := actual.Values()
	if err != nil || after != level {
		return false, fmt.Errorf("Actions access was not retained (write effects may already have applied)")
	}
	if err := settings.PrepareActionsAccess(actual, level); err != nil {
		return false, fmt.Errorf("verify Actions access applicability (write effects may already have applied): %w", err)
	}
	return true, nil
}
