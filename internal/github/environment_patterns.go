package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func patternPath(repo repository.Repository, name string) string {
	return environmentPath(repo, name) + "/deployment-branch-policies"
}

func decodeEnvironmentPattern(raw map[string]json.RawMessage) (settings.EnvironmentPatternState, error) {
	var state settings.EnvironmentPatternState
	if err := environmentKnownFields(raw, "id", "node_id", "name", "type"); err != nil {
		return state, err
	}
	for key, target := range map[string]any{"id": &state.ID, "name": &state.Pattern.Name, "type": &state.Pattern.Type} {
		if err := requiredEnvironmentJSON(raw, key, target); err != nil {
			return state, err
		}
	}
	desired := []settings.EnvironmentPattern{}
	if _, err := settings.ReconcileEnvironmentPatterns([]settings.EnvironmentPatternState{state}, &desired); err != nil {
		return state, err
	}
	return state, nil
}

func (c *EnvironmentsClient) readPatterns(repo repository.Repository, name string) ([]settings.EnvironmentPatternState, error) {
	states := []settings.EnvironmentPatternState{}
	expected := -1
	for page := 1; ; page++ {
		var raw map[string]json.RawMessage
		if err := c.rest.Get(fmt.Sprintf("%s?per_page=100&page=%d", patternPath(repo, name), page), &raw); err != nil {
			return nil, fmt.Errorf("list environment deployment patterns: %w", err)
		}
		var total int
		var entries []map[string]json.RawMessage
		if err := requiredEnvironmentJSON(raw, "total_count", &total); err != nil {
			return nil, err
		}
		if err := requiredEnvironmentJSON(raw, "branch_policies", &entries); err != nil {
			return nil, err
		}
		if total < 0 || (expected >= 0 && expected != total) {
			return nil, fmt.Errorf("deployment pattern inventory count changed or invalid")
		}
		expected = total
		for _, entry := range entries {
			state, err := decodeObservedEnvironmentPattern(entry)
			if err != nil {
				return nil, err
			}
			states = append(states, state)
		}
		if err := settings.ValidateEnvironmentPatternObservation(states); err != nil {
			return nil, err
		}
		if len(states) > total {
			return nil, fmt.Errorf("deployment pattern inventory exceeds count")
		}
		if len(states) == total {
			return states, nil
		}
		if len(entries) != 100 {
			return nil, fmt.Errorf("incomplete deployment pattern inventory")
		}
	}
}

func (c *EnvironmentsClient) applyPatterns(repo repository.Repository, name string, current []settings.EnvironmentPatternState, desired *[]settings.EnvironmentPattern) (bool, error) {
	attempted := false
	changes, err := settings.ReconcileEnvironmentPatterns(current, desired)
	if err != nil {
		return attempted, err
	}
	for _, change := range changes {
		switch change.Operation {
		case "create":
			body, err := json.Marshal(change.Pattern)
			if err != nil {
				return attempted, err
			}
			var raw map[string]json.RawMessage
			attempted = true
			if err := c.rest.Post(patternPath(repo, name), bytes.NewReader(body), &raw); err != nil {
				return attempted, fmt.Errorf("create %s pattern %q (write may have applied): %w", change.Pattern.Type, change.Pattern.Name, err)
			}
			actual, err := decodeEnvironmentPattern(raw)
			if err != nil {
				return attempted, fmt.Errorf("verify created pattern (write may have applied): %w", err)
			}
			if actual.Pattern != change.Pattern {
				return attempted, fmt.Errorf("created pattern differs from request (write may have applied)")
			}
		case "delete":
			path := patternPath(repo, name) + "/" + strconv.FormatInt(change.ID, 10)
			var raw map[string]json.RawMessage
			if err := c.rest.Get(path, &raw); err != nil {
				return attempted, fmt.Errorf("read pattern before removal: %w", err)
			}
			actual, err := decodeObservedEnvironmentPattern(raw)
			if err != nil {
				return attempted, err
			}
			identity, err := actual.RawPattern()
			if err != nil {
				return attempted, err
			}
			if actual.ID != change.ID || identity != change.Pattern {
				return attempted, fmt.Errorf("deployment pattern identity changed before removal")
			}
			attempted = true
			if err := c.rest.Delete(path, nil); err != nil {
				return attempted, fmt.Errorf("remove pattern %d (write may have applied): %w", change.ID, err)
			}
		}
	}
	return attempted, nil
}
