package github

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

const classicDeleteMutation = `mutation ClassicProtectionDelete($input: DeleteBranchProtectionRuleInput!) { deleteBranchProtectionRule(input:$input) { __typename } }`

func (c *ClassicProtectionClient) Delete(repo repository.Repository, candidate settings.ClassicPruneCandidate) error {
	pre := func(err error) error { return &classicPreWriteFailure{err} }
	if candidate.ID() <= 0 || candidate.NodeID() == "" {
		return pre(fmt.Errorf("missing classic deletion identity"))
	}
	fresh, target, err := c.readOne(repo, candidate.NodeID())
	if err != nil {
		return pre(err)
	}
	if err := candidate.Verify(fresh, target.id, target.archived); err != nil {
		return pre(err)
	}
	var response json.RawMessage
	if err := c.graphql.Do(classicDeleteMutation, map[string]any{"input": map[string]any{"branchProtectionRuleId": candidate.NodeID()}}, &response); err != nil {
		return fmt.Errorf("delete classic rule (write may have applied): %w", err)
	}
	ack, err := classicObject(response, "deleteBranchProtectionRule")
	if err != nil {
		return fmt.Errorf("invalid classic deletion acknowledgment (write may have applied): %w", err)
	}
	payload, err := classicObject(ack["deleteBranchProtectionRule"], "__typename")
	if err != nil {
		return fmt.Errorf("invalid classic deletion payload (write may have applied): %w", err)
	}
	typename, err := accessString(payload, "__typename")
	if err != nil || typename != "DeleteBranchProtectionRulePayload" {
		return fmt.Errorf("unconfirmed classic deletion payload type (write may have applied)")
	}
	remaining, err := c.Read(repo)
	if err != nil {
		return fmt.Errorf("verify classic deletion inventory (write may have applied): %w", err)
	}
	if err := candidate.VerifyTarget(remaining.RepositoryID, remaining.Archived, remaining.IsFork); err != nil {
		return fmt.Errorf("verify classic deletion target (write may have applied): %w", err)
	}
	for _, state := range remaining.Rules {
		if state.NodeID == candidate.NodeID() || *state.Rule.ID == candidate.ID() {
			return fmt.Errorf("classic deletion inventory retained selected identity (write may have applied)")
		}
	}
	return nil
}
