package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type ForkWorkflowsClient struct{ rest *api.RESTClient }
type forkWorkflowsPreWriteFailure struct{ cause error }

func (e *forkWorkflowsPreWriteFailure) Error() string        { return e.cause.Error() }
func (e *forkWorkflowsPreWriteFailure) Unwrap() error        { return e.cause }
func (*forkWorkflowsPreWriteFailure) ReadFailedBeforeWrite() {}
func NewForkWorkflowsClient(host string) (*ForkWorkflowsClient, error) {
	c, err := newActionsPolicyRESTClient(host)
	if err != nil {
		return nil, &forkWorkflowsPreWriteFailure{err}
	}
	return &ForkWorkflowsClient{c}, nil
}
func (c *ForkWorkflowsClient) requireVisibility(repo repository.Repository, want string) error {
	var fields map[string]json.RawMessage
	if err := c.rest.Get("repos/"+repo.Owner+"/"+repo.Name, &fields); err != nil {
		return fmt.Errorf("read fork workflow applicability: %w", err)
	}
	var visibility *string
	if json.Unmarshal(fields["visibility"], &visibility) != nil || visibility == nil || (*visibility != "public" && *visibility != "private" && *visibility != "internal") {
		return fmt.Errorf("missing/null or invalid repository visibility for fork workflows")
	}
	if *visibility != want {
		return fmt.Errorf("fork workflow policy requires %s repository visibility; observed %s", want, *visibility)
	}
	return nil
}
func forkApprovalPath(repo repository.Repository) string {
	return actionsPolicyPath(repo) + "/fork-pr-contributor-approval"
}
func privateForkPath(repo repository.Repository) string {
	return actionsPolicyPath(repo) + "/fork-pr-workflows-private-repos"
}
func (c *ForkWorkflowsClient) ReadApproval(repo repository.Repository) (settings.ForkApprovalObservation, error) {
	if err := c.requireVisibility(repo, "public"); err != nil {
		return settings.ForkApprovalObservation{}, err
	}
	var fields map[string]json.RawMessage
	if err := c.rest.Get(forkApprovalPath(repo), &fields); err != nil {
		return settings.ForkApprovalObservation{}, fmt.Errorf("read fork approval policy: %w", err)
	}
	var v *string
	if json.Unmarshal(fields["approval_policy"], &v) != nil || v == nil {
		return settings.ForkApprovalObservation{}, fmt.Errorf("missing/null or invalid fork approval_policy")
	}
	return settings.NewForkApprovalObservation(*v)
}
func (c *ForkWorkflowsClient) ReadPrivate(repo repository.Repository) (settings.PrivateForkObservation, error) {
	if err := c.requireVisibility(repo, "private"); err != nil {
		return settings.PrivateForkObservation{}, err
	}
	var fields map[string]json.RawMessage
	if err := c.rest.Get(privateForkPath(repo), &fields); err != nil {
		return settings.PrivateForkObservation{}, fmt.Errorf("read private fork policy: %w", err)
	}
	var values [4]bool
	for i, key := range settings.PrivateForkKeys() {
		v, err := actionsBool(fields, key)
		if err != nil {
			return settings.PrivateForkObservation{}, err
		}
		values[i] = v
	}
	return settings.NewPrivateForkObservation(values), nil
}
func (c *ForkWorkflowsClient) UpdateApproval(repo repository.Repository, policy string) (bool, error) {
	if !settings.ValidForkApprovalPolicy(policy) {
		return false, &forkWorkflowsPreWriteFailure{fmt.Errorf("invalid fork approval policy")}
	}
	o, err := c.ReadApproval(repo)
	if err != nil {
		return false, &forkWorkflowsPreWriteFailure{err}
	}
	v, _ := o.Value()
	if v == policy {
		return false, nil
	}
	payload, _ := json.Marshal(map[string]string{"approval_policy": policy})
	if err := c.rest.Put(forkApprovalPath(repo), bytes.NewReader(payload), nil); err != nil {
		return false, fmt.Errorf("write fork approval policy (write effects may already have applied): %w", err)
	}
	a, err := c.ReadApproval(repo)
	if err != nil {
		return false, fmt.Errorf("verify fork approval policy (write effects may already have applied): %w", err)
	}
	actual, err := a.Value()
	if err != nil || actual != policy {
		return false, fmt.Errorf("fork approval policy was not retained (write effects may already have applied)")
	}
	return true, nil
}
func (c *ForkWorkflowsClient) UpdatePrivate(repo repository.Repository, d settings.PrivateForkDeclaration) (bool, error) {
	if d.Empty() {
		return false, nil
	}
	o, err := c.ReadPrivate(repo)
	if err != nil {
		return false, &forkWorkflowsPreWriteFailure{err}
	}
	_, changes, err := settings.ComparePrivateFork(o, d)
	if err != nil {
		return false, &forkWorkflowsPreWriteFailure{err}
	}
	if len(changes) == 0 {
		return false, nil
	}
	expected, _ := o.Values()
	payload := map[string]bool{}
	for i, p := range d.Values() {
		if p != nil {
			payload[settings.PrivateForkKeys()[i]] = *p
			expected[i] = *p
		}
	}
	data, _ := json.Marshal(payload)
	if err := c.rest.Put(privateForkPath(repo), bytes.NewReader(data), nil); err != nil {
		return false, fmt.Errorf("write private fork policy (write effects may already have applied): %w", err)
	}
	a, err := c.ReadPrivate(repo)
	if err != nil {
		return false, fmt.Errorf("verify private fork policy (write effects may already have applied): %w", err)
	}
	actual, err := a.Values()
	if err != nil || actual != expected {
		return false, fmt.Errorf("private fork declared values or unowned companions were not retained (write effects may already have applied)")
	}
	return true, nil
}
