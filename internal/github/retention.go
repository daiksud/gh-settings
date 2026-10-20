package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type RetentionClient struct{ rest *api.RESTClient }
type retentionPreWriteFailure struct{ cause error }

func (e *retentionPreWriteFailure) Error() string        { return e.cause.Error() }
func (e *retentionPreWriteFailure) Unwrap() error        { return e.cause }
func (*retentionPreWriteFailure) ReadFailedBeforeWrite() {}
func NewRetentionClient(host string) (*RetentionClient, error) {
	client, err := newActionsPolicyRESTClient(host)
	if err != nil {
		return nil, &retentionPreWriteFailure{fmt.Errorf("configure retention API: %w", err)}
	}
	return &RetentionClient{rest: client}, nil
}
func retentionPath(repo repository.Repository) string {
	return actionsPolicyPath(repo) + "/artifact-and-log-retention"
}
func (c *RetentionClient) Read(repo repository.Repository) (settings.RetentionObservation, error) {
	var fields map[string]json.RawMessage
	if err := c.rest.Get(retentionPath(repo), &fields); err != nil {
		return settings.RetentionObservation{}, fmt.Errorf("read repository retention: %w", err)
	}
	values := make([]int, 0, 2)
	for _, key := range []string{"days", "maximum_allowed_days"} {
		var value *int
		if json.Unmarshal(fields[key], &value) != nil || value == nil {
			return settings.RetentionObservation{}, fmt.Errorf("GitHub returned missing/null or invalid retention %s", key)
		}
		values = append(values, *value)
	}
	observed, err := settings.NewRetentionObservation(values[0], values[1])
	if err != nil {
		return settings.RetentionObservation{}, fmt.Errorf("GitHub returned invalid retention observation: %w", err)
	}
	return observed, nil
}
func (c *RetentionClient) Update(repo repository.Repository, days int) (bool, error) {
	if err := settings.ValidateRetentionDays(days); err != nil {
		return false, &retentionPreWriteFailure{err}
	}
	current, err := c.Read(repo)
	if err != nil {
		return false, &retentionPreWriteFailure{err}
	}
	if err := settings.PrepareRetention(current, days); err != nil {
		return false, &retentionPreWriteFailure{err}
	}
	before, _, _ := current.Values()
	if before == days {
		return false, nil
	}
	payload, err := json.Marshal(map[string]int{"days": days})
	if err != nil {
		return false, &retentionPreWriteFailure{err}
	}
	if err := c.rest.Put(retentionPath(repo), bytes.NewReader(payload), nil); err != nil {
		return false, fmt.Errorf("write retention (write effects may already have applied): %w", err)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify retention (write effects may already have applied): %w", err)
	}
	after, _, err := actual.Values()
	if err != nil {
		return false, fmt.Errorf("verify retention (write effects may already have applied): %w", err)
	}
	if after != days {
		return false, fmt.Errorf("GitHub did not retain requested retention days (write effects may already have applied)")
	}
	if err := settings.PrepareRetention(actual, days); err != nil {
		return false, fmt.Errorf("verify retention against current upper policy (write effects may already have applied): %w", err)
	}
	return true, nil
}
