package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func (c *Client) ReadVariables(repo repository.Repository) (settings.VariableInventory, error) {
	return c.readVariableInventory(variablesPath(repo))
}
func (c *Client) ReadAgentVariables(repo repository.Repository) (settings.VariableInventory, error) {
	return c.readVariableInventory(agentVariablesPath(repo))
}
func (c *Client) ReadEnvironmentVariables(repo repository.Repository, environment string) (settings.VariableInventory, error) {
	path, id, err := c.environmentVariableScope(repo, environment)
	if err != nil {
		return settings.VariableInventory{}, err
	}
	inventory, err := c.readVariableInventory(path)
	if err != nil {
		return settings.VariableInventory{}, err
	}
	values, err := inventory.Values()
	if err != nil {
		return settings.VariableInventory{}, err
	}
	_, again, err := c.environmentVariableScope(repo, environment)
	if err != nil || again != id {
		return settings.VariableInventory{}, fmt.Errorf("Environment identity changed during variable inventory acquisition")
	}
	return settings.NewEnvironmentVariableInventory(environment, id, values)
}

// Each requested page contains at most 30 variable values, each potentially
// large and JSON-escaped. Bound acquisition without accepting a partial page.
const variableInventoryPageLimit = 16 << 20

// Unknown response metadata remains unowned, but aliases of selected keys
// cannot coexist with or replace the canonical field.
func variableInventoryAliases(fields map[string]json.RawMessage, canonical ...string) bool {
	for name := range fields {
		for _, key := range canonical {
			if name != key && strings.EqualFold(name, key) {
				return true
			}
		}
	}
	return false
}

// We request 30 items per page. Require a stable total, every name/value, no
// duplicates, and exactly the promised complete count.
func (c *Client) readVariableInventory(path string) (settings.VariableInventory, error) {
	values := []settings.RepositoryVariable{}
	expected := -1
	for page := 1; ; page++ {
		response, err := c.rest.Request(http.MethodGet, fmt.Sprintf("%s?per_page=30&page=%d", path, page), nil)
		if err != nil {
			return settings.VariableInventory{}, fmt.Errorf("read variable inventory page %d failed", page)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, variableInventoryPageLimit+1))
		response.Body.Close()
		links := response.Header.Values("Link")
		for _, link := range links {
			if strings.TrimSpace(link) == "" {
				return settings.VariableInventory{}, fmt.Errorf("invalid variable inventory pagination")
			}
		}
		next, linkErr := webhookHasNext(strings.Join(links, ","))
		var fields map[string]json.RawMessage
		if response.StatusCode != http.StatusOK || readErr != nil || linkErr != nil || len(body) > variableInventoryPageLimit || !validVariableJSONUnicode(body) || webhookJSON(body, &fields) != nil || fields == nil || variableInventoryAliases(fields, "total_count", "variables") {
			return settings.VariableInventory{}, fmt.Errorf("incomplete or invalid variable inventory page %d", page)
		}
		var total *int
		var entries []map[string]json.RawMessage
		if json.Unmarshal(fields["total_count"], &total) != nil || total == nil || *total < 0 || json.Unmarshal(fields["variables"], &entries) != nil || entries == nil || len(entries) > 30 {
			return settings.VariableInventory{}, fmt.Errorf("incomplete or invalid variable inventory page %d", page)
		}
		if expected < 0 {
			expected = *total
		} else if expected != *total {
			return settings.VariableInventory{}, fmt.Errorf("variable inventory count changed during pagination")
		}
		for _, item := range entries {
			var name, value *string
			if item == nil || variableInventoryAliases(item, "name", "value") || json.Unmarshal(item["name"], &name) != nil || name == nil || json.Unmarshal(item["value"], &value) != nil || value == nil {
				return settings.VariableInventory{}, fmt.Errorf("incomplete variable inventory entry")
			}
			values = append(values, settings.RepositoryVariable{Name: *name, Value: *value})
		}
		if len(values) > expected {
			return settings.VariableInventory{}, fmt.Errorf("variable inventory exceeds total count")
		}
		if len(values) == expected {
			if next {
				return settings.VariableInventory{}, fmt.Errorf("variable inventory pagination contradicts total count")
			}
			return settings.NewVariableInventory(values)
		}
		if len(links) > 0 && !next {
			return settings.VariableInventory{}, fmt.Errorf("variable inventory pagination contradicts total count")
		}
		if len(entries) != 30 {
			return settings.VariableInventory{}, fmt.Errorf("incomplete variable inventory before total count")
		}
	}
}

func (c *Client) DeleteVariable(repo repository.Repository, change settings.VariableChange) error {
	return c.deleteVariable(variablesPath(repo), change)
}
func (c *Client) DeleteAgentVariable(repo repository.Repository, change settings.VariableChange) error {
	return c.deleteVariable(agentVariablesPath(repo), change)
}
func (c *Client) DeleteEnvironmentVariable(repo repository.Repository, environment string, id int64, change settings.VariableChange) error {
	path, actual, err := c.environmentVariableScope(repo, environment)
	if err != nil {
		return &variablePreWriteFailure{err}
	}
	if id <= 0 || actual != id {
		return &variablePreWriteFailure{fmt.Errorf("Environment identity changed before variable deletion")}
	}
	if err := c.deleteVariable(path, change); err != nil {
		return err
	}
	_, actual, err = c.environmentVariableScope(repo, environment)
	if err != nil || actual != id {
		return fmt.Errorf("Environment identity unavailable or changed after variable deletion (write may have applied)")
	}
	return nil
}
func (c *Client) deleteVariable(path string, change settings.VariableChange) error {
	before, _ := change.Values()
	if change.Operation() != "delete" || before == nil {
		return &variablePreWriteFailure{fmt.Errorf("expected typed variable deletion")}
	}
	observed, err := c.readVariable(path, change.Name())
	if err != nil {
		return &variablePreWriteFailure{err}
	}
	value, err := observed.Value()
	if err != nil || value == nil || value.Name != change.Name() || value.Value != *before {
		return &variablePreWriteFailure{fmt.Errorf("variable identity or value changed before deletion")}
	}
	response, err := c.rest.Request(http.MethodDelete, path+"/"+change.Name(), nil)
	if err != nil {
		return fmt.Errorf("delete variable %q failed (write may have applied)", change.Name())
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected variable delete status %d (write may have applied)", response.StatusCode)
	}
	inventory, err := c.readVariableInventory(path)
	if err != nil {
		return fmt.Errorf("verify variable deletion failed (write may have applied): %w", err)
	}
	values, err := inventory.Values()
	if err != nil {
		return err
	}
	for _, value := range values {
		if value.Name == change.Name() {
			return fmt.Errorf("variable deletion readback retained name (write may have applied)")
		}
	}
	return nil
}
