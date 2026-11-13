package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func classicEnvironmentSetsEqual(desired []string, acquired *[]string) bool {
	if acquired == nil {
		return false
	}
	a, b := slices.Clone(desired), slices.Clone(*acquired)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

// Only identities are acquired: unrelated Environment policy and secrets are not prerequisites.
func (c *ClassicProtectionClient) classicDeploymentReferences(repo repository.Repository, target classicTarget, names []string) (map[string]string, error) {
	const query = `query ClassicDeploymentEnvironments($owner:String!,$name:String!,$cursor:String){repository(owner:$owner,name:$name,followRenames:false){` + classicTargetFields + ` environments(first:100,after:$cursor){totalCount pageInfo{hasNextPage endCursor} nodes{id databaseId name}}}}`
	inventory, ids, cursors := map[string]string{}, map[string]bool{}, map[string]bool{}
	cursor, expected := "", -1
	for {
		var after any
		if cursor != "" {
			after = cursor
		}
		var response json.RawMessage
		if err := c.graphql.Do(query, map[string]any{"owner": repo.Owner, "name": repo.Name, "cursor": after}, &response); err != nil {
			// This identity-only lookup must not expose upstream bodies, transport
			// errors or response-controlled member names through apply diagnostics.
			return nil, fmt.Errorf("read classic deployment references: request failed")
		}
		top, err := classicObject(response, "repository")
		if err != nil {
			return nil, fmt.Errorf("invalid classic Environment repository response")
		}
		fields, err := accessObject(top["repository"], "classic Environment repository")
		if err != nil {
			return nil, fmt.Errorf("invalid classic Environment repository object")
		}
		connection, ok := fields["environments"]
		if !ok {
			return nil, fmt.Errorf("missing classic Environment inventory")
		}
		delete(fields, "environments")
		raw, _ := json.Marshal(fields)
		current, err := decodeClassicTarget(raw, repo)
		if err != nil {
			return nil, fmt.Errorf("incomplete or unsupported classic Environment repository identity")
		}
		if current.id != target.id || current.archived != target.archived {
			return nil, fmt.Errorf("classic Environment target identity or archive state changed")
		}
		nodes, total, end, next, err := classicPage(connection)
		if err != nil {
			return nil, fmt.Errorf("incomplete classic Environment inventory page")
		}
		if expected >= 0 && expected != total {
			return nil, fmt.Errorf("classic Environment inventory count changed")
		}
		expected = total
		for _, raw := range nodes {
			fields, err := classicObject(raw, "id", "databaseId", "name")
			if err != nil {
				return nil, fmt.Errorf("incomplete classic Environment identity object")
			}
			id, err := accessString(fields, "id")
			if err != nil {
				return nil, fmt.Errorf("invalid classic Environment node identity")
			}
			name, err := accessString(fields, "name")
			if err != nil {
				return nil, fmt.Errorf("invalid classic Environment name")
			}
			// The schema permits null; positive numeric identity is supplementary only.
			if !bytes.Equal(bytes.TrimSpace(fields["databaseId"]), []byte("null")) {
				if _, err := classicInt(fields, "databaseId"); err != nil {
					return nil, fmt.Errorf("invalid classic Environment numeric identity")
				}
			}
			if ids[id] || inventory[name] != "" {
				return nil, fmt.Errorf("ambiguous classic Environment identity")
			}
			ids[id], inventory[name] = true, id
		}
		if len(inventory) > total || !next && len(inventory) != total {
			return nil, fmt.Errorf("incomplete classic Environment inventory")
		}
		if !next {
			break
		}
		if cursors[end] {
			return nil, fmt.Errorf("repeated classic Environment cursor")
		}
		cursors[end], cursor = true, end
	}
	selected := make(map[string]string, len(names))
	for _, name := range names {
		id, ok := inventory[name]
		if !ok {
			return nil, fmt.Errorf("classic deployment Environment reference is missing or does not use exact acquired spelling")
		}
		selected[name] = id
	}
	return selected, nil
}
