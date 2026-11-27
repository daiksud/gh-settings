package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// CopilotInstructionsClient stores only the repository instructions toggle.
// These settings do not enable Copilot review or grant access to it.
type CopilotInstructionsClient struct{ rest *api.RESTClient }

func NewCopilotInstructionsClient(host string) (*CopilotInstructionsClient, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &copilotInstructionsPreWriteFailure{fmt.Errorf("configure Copilot custom instructions API")}
	}
	return &CopilotInstructionsClient{rest: rest}, nil
}
func copilotInstructionsPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/copilot/code-review/settings"
}

type copilotInstructionsState struct {
	Enabled      *bool                 `json:"custom_instructions_enabled"`
	PublicSearch optionalSearchSetting `json:"cross_repo_context_enabled"`
}

func (c *CopilotInstructionsClient) Read(repo repository.Repository) (settings.CopilotInstructionsObservation, error) {
	state, err := c.read(repo)
	if err != nil {
		return settings.CopilotInstructionsObservation{}, err
	}
	return settings.NewCopilotInstructionsObservation(*state.Enabled), nil
}

func (c *CopilotInstructionsClient) read(repo repository.Repository) (copilotInstructionsState, error) {
	response, err := c.rest.Request(http.MethodGet, copilotInstructionsPath(repo), nil)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) {
			return copilotInstructionsState{}, fmt.Errorf("read Copilot custom instructions: HTTP %d", httpErr.StatusCode)
		}
		return copilotInstructionsState{}, fmt.Errorf("read Copilot custom instructions failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return copilotInstructionsState{}, fmt.Errorf("unexpected Copilot custom instructions read status %d", response.StatusCode)
	}
	var raw copilotInstructionsState
	if decodeCopilotSettings(response.Body, &raw) != nil || raw.Enabled == nil {
		return copilotInstructionsState{}, fmt.Errorf("missing/null or invalid Copilot custom instructions boolean")
	}
	return raw, nil
}

// copilotInstructionsPreWriteFailure confirms no PATCH was attempted.
type copilotInstructionsPreWriteFailure struct{ error }

func (*copilotInstructionsPreWriteFailure) ReadFailedBeforeWrite() {}

// Update returns true only after a mutation and matching independent readback.
// Errors after PATCH have uncertain effects; false does not mean rollback.
func (c *CopilotInstructionsClient) Update(repo repository.Repository, desired bool) (bool, error) {
	before, err := c.read(repo)
	if err != nil {
		return false, &copilotInstructionsPreWriteFailure{err}
	}
	if *before.Enabled == desired {
		return false, nil
	}
	body, _ := json.Marshal(map[string]bool{"custom_instructions_enabled": desired})
	response, err := c.rest.Request(http.MethodPatch, copilotInstructionsPath(repo), bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("write Copilot custom instructions failed (write effects may already have applied)")
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected Copilot custom instructions write status %d (write effects may already have applied)", response.StatusCode)
	}
	after, err := c.read(repo)
	if err != nil {
		return false, fmt.Errorf("verify Copilot custom instructions failed (write effects may already have applied)")
	}
	if *after.Enabled != desired || !reflect.DeepEqual(before.PublicSearch, after.PublicSearch) {
		return false, fmt.Errorf("Copilot custom instructions was not retained (write effects may already have applied)")
	}
	return true, nil
}
