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
	var raw struct {
		ID   *int64  `json:"id"`
		Name *string `json:"name"`
	}
	decoder := json.NewDecoder(response.Body)
	if decoder.Decode(&raw) != nil || raw.ID == nil || *raw.ID <= 0 || raw.Name == nil {
		return "", 0, fmt.Errorf("incomplete or invalid selected environment identity")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return "", 0, fmt.Errorf("invalid trailing selected environment response")
	}
	if settings.EnvironmentIdentity(*raw.Name) != settings.EnvironmentIdentity(environment) {
		return "", 0, fmt.Errorf("selected environment response identity mismatch")
	}
	return path + "/variables", *raw.ID, nil
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
