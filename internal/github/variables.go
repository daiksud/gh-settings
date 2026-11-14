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
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// NewVariablesClient pins the Variables contract, rejects redirects and avoids
// logging plaintext variable bodies when GH_DEBUG enables HTTP tracing.
func NewVariablesClient(host string) (*Client, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, LogIgnoreEnv: true, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, fmt.Errorf("configure repository Variables API")
	}
	return &Client{rest: rest}, nil
}
func variablesPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/actions/variables"
}

func (c *Client) ReadVariable(repo repository.Repository, name string) (settings.VariableObservation, error) {
	return c.readVariable(variablesPath(repo), name)
}

func agentVariablesPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/agents/variables"
}

func (c *Client) ReadAgentVariable(repo repository.Repository, name string) (settings.VariableObservation, error) {
	return c.readVariable(agentVariablesPath(repo), name)
}

func (c *Client) readVariable(path, name string) (settings.VariableObservation, error) {
	canonical, err := settings.VariableName(name)
	if err != nil {
		return settings.VariableObservation{}, err
	}
	response, err := c.rest.Request(http.MethodGet, path+"/"+canonical, nil)
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
	const limit = 1 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	var fields map[string]json.RawMessage
	if err != nil || len(body) > limit || !validVariableJSONUnicode(body) || webhookJSON(body, &fields) != nil || fields == nil {
		return settings.VariableObservation{}, fmt.Errorf("incomplete or invalid repository variable response")
	}
	// Only exact spellings may supply identity or value. In particular, Go's
	// struct decoder would accept case aliases as the same field.
	for key := range fields {
		if key != "name" && strings.EqualFold(key, "name") || key != "value" && strings.EqualFold(key, "value") {
			return settings.VariableObservation{}, fmt.Errorf("incomplete or invalid repository variable response")
		}
	}
	var observedName, observedValue *string
	if json.Unmarshal(fields["name"], &observedName) != nil || observedName == nil || json.Unmarshal(fields["value"], &observedValue) != nil || observedValue == nil {
		return settings.VariableObservation{}, fmt.Errorf("incomplete or invalid repository variable response")
	}
	actual, err := settings.VariableName(*observedName)
	if err != nil || actual != canonical {
		return settings.VariableObservation{}, fmt.Errorf("repository variable response identity mismatch")
	}
	return settings.NewVariableObservation(settings.RepositoryVariable{Name: canonical, Value: *observedValue})
}

// encoding/json replaces malformed UTF-8 and unpaired UTF-16 escape values
// with U+FFFD. Such normalization cannot establish exact selected equality.
func validVariableJSONUnicode(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	inString := false
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || data[i] != '\\' {
			continue
		}
		if i+1 >= len(data) {
			return false
		}
		i++
		if data[i] != 'u' {
			continue
		}
		if i+4 >= len(data) {
			return false
		}
		value, err := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if value >= 0xdc00 && value <= 0xdfff {
			return false
		}
		if value >= 0xd800 && value <= 0xdbff {
			if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}

// variablePreWriteFailure establishes that no variable mutation was attempted.
type variablePreWriteFailure struct{ error }

func (*variablePreWriteFailure) ReadFailedBeforeWrite() {}

func (c *Client) UpsertVariable(repo repository.Repository, desired settings.RepositoryVariable) (bool, error) {
	return c.upsertVariable(variablesPath(repo), desired)
}

func (c *Client) UpsertAgentVariable(repo repository.Repository, desired settings.RepositoryVariable) (bool, error) {
	return c.upsertVariable(agentVariablesPath(repo), desired)
}

func (c *Client) upsertVariable(collectionPath string, desired settings.RepositoryVariable) (bool, error) {
	validated, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{desired})
	if err != nil {
		return false, &variablePreWriteFailure{err}
	}
	desired = validated.Values()[0]
	observation, err := c.readVariable(collectionPath, desired.Name)
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
	method, path, status := http.MethodPost, collectionPath, http.StatusCreated
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
	actual, err := c.readVariable(collectionPath, desired.Name)
	if err != nil {
		return false, fmt.Errorf("verify repository variable %q failed (write effects may already have applied)", desired.Name)
	}
	after, err := actual.Value()
	if err != nil || after == nil || *after != desired {
		return false, fmt.Errorf("repository variable %q was not retained (write effects may already have applied)", desired.Name)
	}
	return true, nil
}
