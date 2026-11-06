package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type CacheRetentionClient struct{ rest *api.RESTClient }
type CacheStorageClient struct{ rest *api.RESTClient }
type actionsGeneralPreWriteFailure struct{ cause error }

func (e *actionsGeneralPreWriteFailure) Error() string        { return e.cause.Error() }
func (e *actionsGeneralPreWriteFailure) Unwrap() error        { return e.cause }
func (*actionsGeneralPreWriteFailure) ReadFailedBeforeWrite() {}

func NewCacheRetentionClient(host string) (*CacheRetentionClient, error) {
	c, err := newActionsPolicyRESTClient(host)
	if err != nil {
		return nil, &actionsGeneralPreWriteFailure{err}
	}
	return &CacheRetentionClient{rest: c}, nil
}
func NewCacheStorageClient(host string) (*CacheStorageClient, error) {
	c, err := newActionsPolicyRESTClient(host)
	if err != nil {
		return nil, &actionsGeneralPreWriteFailure{err}
	}
	return &CacheStorageClient{rest: c}, nil
}
func actionsCachePath(repo repository.Repository, limit string) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/actions/cache/" + limit + "-limit"
}
func readActionsInteger(c *api.RESTClient, path, key string) (int, error) {
	var fields map[string]json.RawMessage
	if err := c.Get(path, &fields); err != nil {
		return 0, fmt.Errorf("read Actions %s: %w", key, err)
	}
	var value *int
	if json.Unmarshal(fields[key], &value) != nil || value == nil {
		return 0, fmt.Errorf("GitHub returned missing/null or invalid Actions %s", key)
	}
	return *value, nil
}
func readActionsVisibility(c *api.RESTClient, repo repository.Repository) (string, map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := c.Get("repos/"+repo.Owner+"/"+repo.Name, &fields); err != nil {
		return "", nil, fmt.Errorf("read Actions repository applicability: %w", err)
	}
	var fullName *string
	if json.Unmarshal(fields["full_name"], &fullName) != nil || fullName == nil || !strings.EqualFold(*fullName, repo.Owner+"/"+repo.Name) {
		return "", nil, fmt.Errorf("missing/null or mismatched repository identity for Actions")
	}
	var visibility *string
	if json.Unmarshal(fields["visibility"], &visibility) != nil || visibility == nil || (*visibility != "public" && *visibility != "private" && *visibility != "internal") {
		return "", nil, fmt.Errorf("missing/null or invalid repository visibility for Actions")
	}
	return *visibility, fields, nil
}
func (c *CacheRetentionClient) Read(repo repository.Repository) (settings.CacheRetentionObservation, error) {
	visibility, _, err := readActionsVisibility(c.rest, repo)
	if err != nil {
		return settings.CacheRetentionObservation{}, err
	}
	days, err := readActionsInteger(c.rest, actionsCachePath(repo, "retention"), "max_cache_retention_days")
	if err != nil {
		return settings.CacheRetentionObservation{}, err
	}
	return settings.NewCacheRetentionObservation(days, visibility)
}
func (c *CacheStorageClient) Read(repo repository.Repository) (settings.CacheStorageObservation, error) {
	gb, err := readActionsInteger(c.rest, actionsCachePath(repo, "storage"), "max_cache_size_gb")
	if err != nil {
		return settings.CacheStorageObservation{}, err
	}
	return settings.NewCacheStorageObservation(gb)
}
func (c *CacheRetentionClient) Update(repo repository.Repository, days int) (bool, error) {
	if err := settings.ValidateCacheRetentionDays(days); err != nil {
		return false, &actionsGeneralPreWriteFailure{err}
	}
	current, err := c.Read(repo)
	if err != nil {
		return false, &actionsGeneralPreWriteFailure{err}
	}
	if err := settings.PrepareCacheRetention(current, days); err != nil {
		return false, &actionsGeneralPreWriteFailure{err}
	}
	before, _, _ := current.Values()
	if before == days {
		return false, nil
	}
	payload, _ := json.Marshal(map[string]int{"max_cache_retention_days": days})
	if err := c.rest.Put(actionsCachePath(repo, "retention"), bytes.NewReader(payload), nil); err != nil {
		return false, fmt.Errorf("write cache retention (write effects may already have applied): %w", err)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify cache retention (write effects may already have applied): %w", err)
	}
	after, _, err := actual.Values()
	if err != nil || after != days {
		return false, fmt.Errorf("cache retention was not retained (write effects may already have applied)")
	}
	if err := settings.PrepareCacheRetention(actual, days); err != nil {
		return false, fmt.Errorf("verify cache retention applicability (write effects may already have applied): %w", err)
	}
	return true, nil
}
func (c *CacheStorageClient) Update(repo repository.Repository, gb int) (bool, error) {
	if err := settings.ValidateCacheSizeGB(gb); err != nil {
		return false, &actionsGeneralPreWriteFailure{err}
	}
	current, err := c.Read(repo)
	if err != nil {
		return false, &actionsGeneralPreWriteFailure{err}
	}
	before, _ := current.Value()
	if before == gb {
		return false, nil
	}
	payload, _ := json.Marshal(map[string]int{"max_cache_size_gb": gb})
	if err := c.rest.Put(actionsCachePath(repo, "storage"), bytes.NewReader(payload), nil); err != nil {
		return false, fmt.Errorf("write cache storage (write effects may already have applied): %w", err)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify cache storage (write effects may already have applied): %w", err)
	}
	after, err := actual.Value()
	if err != nil || after != gb {
		return false, fmt.Errorf("cache storage was not retained (write effects may already have applied)")
	}
	return true, nil
}
