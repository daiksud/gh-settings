package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// environmentVariablesPath verifies the selected Environment exists without
// acquiring or taking ownership of its protection settings. In particular, an
// Environment 404 must not become an absent variable followed by implicit creation.
func (c *Client) environmentVariablesPath(repo repository.Repository, environment string) (string, error) {
	path, _, err := c.environmentVariableScope(repo, environment)
	return path, err
}
func (c *Client) environmentVariableScope(repo repository.Repository, environment string) (string, int64, error) {
	if _, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: environment}}); err != nil {
		return "", 0, err
	}
	path := environmentPath(repo, environment)
	response, err := c.rest.Request(http.MethodGet, path, nil)
	if err != nil {
		return "", 0, fmt.Errorf("read selected environment %q failed (unavailable is not absent)", environment)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("unexpected selected environment read status %d", response.StatusCode)
	}
	// An Environment detail can include large, unowned reviewer profiles. Do
	// not interpret duplicate members or Go's case-insensitive struct aliases
	// as complete evidence for the parent selected by name.
	data, err := io.ReadAll(io.LimitReader(response.Body, environmentResponseLimit+1))
	var raw map[string]json.RawMessage
	if err != nil || len(data) > environmentResponseLimit || environmentJSON(data, &raw) != nil || raw == nil {
		return "", 0, fmt.Errorf("incomplete or invalid selected environment identity")
	}
	if rejectEnvironmentAliases(raw, "id", "name") != nil {
		return "", 0, fmt.Errorf("ambiguous selected environment identity")
	}
	var id *int64
	var name *string
	if json.Unmarshal(raw["id"], &id) != nil || id == nil || *id <= 0 || json.Unmarshal(raw["name"], &name) != nil || name == nil {
		return "", 0, fmt.Errorf("incomplete or invalid selected environment identity")
	}
	if settings.EnvironmentIdentity(*name) != settings.EnvironmentIdentity(environment) {
		return "", 0, fmt.Errorf("selected environment response identity mismatch")
	}
	return path + "/variables", *id, nil
}

func (c *Client) ReadEnvironmentVariable(repo repository.Repository, environment, name string) (settings.VariableObservation, error) {
	if _, err := settings.VariableName(name); err != nil {
		return settings.VariableObservation{}, err
	}
	path, err := c.environmentVariablesPath(repo, environment)
	if err != nil {
		return settings.VariableObservation{}, err
	}
	return c.readVariable(path, name)
}

func (c *Client) UpsertEnvironmentVariable(repo repository.Repository, environment string, desired settings.RepositoryVariable) (bool, error) {
	if _, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{desired}); err != nil {
		return false, &variablePreWriteFailure{err}
	}
	path, err := c.environmentVariablesPath(repo, environment)
	if err != nil {
		return false, &variablePreWriteFailure{err}
	}
	return c.upsertVariable(path, desired)
}

// ReadArchivedEnvironmentVariables binds selected values to a stable parent ID.
// These independent GETs do not form an atomic snapshot.
func (c *Client) ReadArchivedEnvironmentVariables(repo repository.Repository, environment string, names []string) (int64, []settings.VariableObservation, error) {
	path, id, err := c.environmentVariableScope(repo, environment)
	if err != nil {
		return 0, nil, fmt.Errorf("read archived environment variable parent failed")
	}
	observed := make([]settings.VariableObservation, 0, len(names))
	for _, name := range names {
		if _, err := settings.VariableName(name); err != nil {
			return 0, nil, fmt.Errorf("invalid selected environment variable")
		}
		value, err := c.readVariable(path, name)
		if err != nil {
			return 0, nil, fmt.Errorf("read archived environment variable failed")
		}
		observed = append(observed, value)
	}
	_, freshID, err := c.environmentVariableScope(repo, environment)
	if err != nil || freshID != id {
		return 0, nil, fmt.Errorf("archived environment variable parent identity changed or unavailable")
	}
	return id, observed, nil
}
