package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type ForkingClient struct{ rest *api.RESTClient }
type forkingPreWriteFailure struct{ error }

func (*forkingPreWriteFailure) ReadFailedBeforeWrite() {}
func NewForkingClient(host string) (*ForkingClient, error) {
	rest, err := newSecurityRESTClient(host)
	if err != nil {
		return nil, &forkingPreWriteFailure{err}
	}
	return &ForkingClient{rest}, nil
}

type forkingRepository struct {
	ID         int64  `json:"id"`
	FullName   string `json:"full_name"`
	Private    *bool  `json:"private"`
	Visibility string `json:"visibility"`
	Fork       *bool  `json:"fork"`
	Archived   *bool  `json:"archived"`
	Allowed    *bool  `json:"allow_forking"`
	Owner      struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"owner"`
}

func (s forkingRepository) observation(repo repository.Repository, requireActive bool) (settings.ForkingObservation, error) {
	if s.Private == nil || !*s.Private || s.Visibility != "private" || s.Fork == nil || *s.Fork || s.Archived == nil || (requireActive && *s.Archived) || s.Owner.Type != "Organization" || !strings.EqualFold(s.Owner.Login, repo.Owner) || s.Allowed == nil {
		if requireActive {
			return settings.ForkingObservation{}, fmt.Errorf("private forking updates require a completely observed active, non-fork, private organization repository")
		}
		return settings.ForkingObservation{}, fmt.Errorf("private forking requires a completely observed non-fork, private organization repository")
	}
	return settings.NewForkingObservation(repo, s.ID, s.Owner.ID, s.FullName, *s.Allowed)
}
func (c *ForkingClient) repository(repo repository.Repository, requireActive bool) (forkingRepository, settings.ForkingObservation, error) {
	var r forkingRepository
	if err := c.rest.Get(securityRepositoryPath(repo), &r); err != nil {
		return r, settings.ForkingObservation{}, fmt.Errorf("read private forking repository: %w", err)
	}
	o, err := r.observation(repo, requireActive)
	return r, o, err
}

// Read never infers a hidden organization policy from metadata access or admin flags.
func (c *ForkingClient) Read(repo repository.Repository) (settings.ForkingObservation, error) {
	return c.read(repo, false)
}

func (c *ForkingClient) read(repo repository.Repository, requireActive bool) (settings.ForkingObservation, error) {
	r, before, err := c.repository(repo, requireActive)
	if err != nil {
		return settings.ForkingObservation{}, err
	}
	var org struct {
		ID      int64  `json:"id"`
		Login   string `json:"login"`
		Allowed *bool  `json:"members_can_fork_private_repositories"`
	}
	if err := c.rest.Get("orgs/"+repo.Owner, &org); err != nil {
		return settings.ForkingObservation{}, fmt.Errorf("read organization private forking policy: %w", err)
	}
	if org.ID != r.Owner.ID || !strings.EqualFold(org.Login, repo.Owner) || org.Allowed == nil || !*org.Allowed {
		return settings.ForkingObservation{}, fmt.Errorf("organization private forking policy is hidden, disabled or belongs to another owner")
	}
	fresh, after, err := c.repository(repo, requireActive)
	if err != nil {
		return settings.ForkingObservation{}, err
	}
	if before != after || *r.Archived != *fresh.Archived {
		return settings.ForkingObservation{}, fmt.Errorf("private forking target changed during observation")
	}
	return before, nil
}

// Update re-observes target and policy, sends one scalar, then checks response and
// independent readback. Name-addressed endpoints do not provide an atomic lock.
func (c *ForkingClient) Update(repo repository.Repository, desired bool) (bool, error) {
	before, err := c.read(repo, true)
	if err != nil {
		return false, &forkingPreWriteFailure{err}
	}
	value, _ := before.Value()
	if value == desired {
		return false, nil
	}
	fresh, err := c.read(repo, true)
	if err != nil {
		return false, &forkingPreWriteFailure{err}
	}
	if before != fresh {
		return false, &forkingPreWriteFailure{fmt.Errorf("private forking state changed before write")}
	}
	body, _ := json.Marshal(struct {
		Allowed bool `json:"allow_forking"`
	}{desired})
	var response forkingRepository
	if err := c.rest.Patch(securityRepositoryPath(repo), bytes.NewReader(body), &response); err != nil {
		return false, fmt.Errorf("update private forking policy (write may have applied): %w", err)
	}
	expected := before.WithAllowed(desired)
	actual, err := response.observation(repo, true)
	if err != nil || actual != expected {
		return false, fmt.Errorf("private forking PATCH response did not confirm target and state; write may have applied")
	}
	actual, err = c.read(repo, true)
	if err != nil {
		return false, fmt.Errorf("private forking readback failed (write may have applied): %w", err)
	}
	if actual != expected {
		return false, fmt.Errorf("private forking readback did not confirm target and state; write may have applied")
	}
	return true, nil
}
