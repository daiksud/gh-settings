package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"io"
	"net/http"
	"time"
)

// NewVariablesClient pins the Variables contract and rejects redirects, using
// existing gh authentication without changing permissions.
func NewVariablesClient(host string) (*Client, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, fmt.Errorf("configure repository Variables API")
	}
	return &Client{rest: rest}, nil
}
func variablesPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/actions/variables"
}

func (c *Client) ReadVariable(repo repository.Repository, name string) (settings.VariableObservation, error) {
	canonical, err := settings.VariableName(name)
	if err != nil {
		return settings.VariableObservation{}, err
	}
	response, err := c.rest.Request(http.MethodGet, variablesPath(repo)+"/"+canonical, nil)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) {
			if httpErr.StatusCode == http.StatusNotFound {
				return settings.NewAbsentVariableObservation(), nil
			}
			return settings.VariableObservation{}, fmt.Errorf("read repository variable %q: HTTP %d", canonical, httpErr.StatusCode)
		}
		return settings.VariableObservation{}, fmt.Errorf("read repository variable %q failed", canonical)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return settings.VariableObservation{}, fmt.Errorf("unexpected repository variable read status %d", response.StatusCode)
	}
	var raw struct {
		Name  *string `json:"name"`
		Value *string `json:"value"`
	}
	decoder := json.NewDecoder(response.Body)
	if decoder.Decode(&raw) != nil || raw.Name == nil || raw.Value == nil {
		return settings.VariableObservation{}, fmt.Errorf("incomplete or invalid repository variable response")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return settings.VariableObservation{}, fmt.Errorf("invalid trailing repository variable response")
	}
	actual, err := settings.VariableName(*raw.Name)
	if err != nil || actual != canonical {
		return settings.VariableObservation{}, fmt.Errorf("repository variable response identity mismatch")
	}
	return settings.NewVariableObservation(settings.RepositoryVariable{Name: canonical, Value: *raw.Value})
}

// variablePreWriteFailure establishes that no variable mutation was attempted.
type variablePreWriteFailure struct{ error }

func (*variablePreWriteFailure) ReadFailedBeforeWrite() {}

func (c *Client) UpsertVariable(repo repository.Repository, desired settings.RepositoryVariable) (bool, error) {
	validated, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{desired})
	if err != nil {
		return false, &variablePreWriteFailure{err}
	}
	desired = validated.Values()[0]
	observation, err := c.ReadVariable(repo, desired.Name)
	if err != nil {
		return false, &variablePreWriteFailure{err}
	}
	before, err := observation.Value()
	if err != nil {
		return false, &variablePreWriteFailure{err}
	}
	if before != nil && before.Value == desired.Value {
		return false, nil
	}
	method, path, status := http.MethodPost, variablesPath(repo), http.StatusCreated
	payload := map[string]string{"value": desired.Value}
	if before == nil {
		payload["name"] = desired.Name
	} else {
		method, path, status = http.MethodPatch, path+"/"+desired.Name, http.StatusNoContent
	}
	body, _ := json.Marshal(payload)
	response, err := c.rest.Request(method, path, bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("write repository variable %q failed (write effects may already have applied)", desired.Name)
	}
	response.Body.Close()
	if response.StatusCode != status {
		return false, fmt.Errorf("unexpected repository variable write status %d (write effects may already have applied)", response.StatusCode)
	}
	actual, err := c.ReadVariable(repo, desired.Name)
	if err != nil {
		return false, fmt.Errorf("verify repository variable %q failed (write effects may already have applied)", desired.Name)
	}
	after, err := actual.Value()
	if err != nil || after == nil || *after != desired {
		return false, fmt.Errorf("repository variable %q was not retained (write effects may already have applied)", desired.Name)
	}
	return true, nil
}
