package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func NewAutolinksClient(host string) (*Client, error) {
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, LogIgnoreEnv: true, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, fmt.Errorf("configure autolink client failed")
	}
	return &Client{rest: rest}, nil
}
func (c *Client) autolinkGET(path string, out any) error {
	response, err := c.rest.Request(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("read autolink state failed")
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	var extra any
	if response.StatusCode != 200 || response.Header.Get("Link") != "" || decoder.Decode(out) != nil || decoder.Decode(&extra) != io.EOF {
		return fmt.Errorf("incomplete autolink response")
	}
	return nil
}
func (c *Client) autolinkTarget(repo repository.Repository) (settings.AutolinkTarget, error) {
	var raw collaboratorRawRepository
	if err := c.autolinkGET("repos/"+repo.Owner+"/"+repo.Name, &raw); err != nil {
		return settings.AutolinkTarget{}, err
	}
	if raw.ID == nil || *raw.ID <= 0 || raw.FullName == nil || !strings.EqualFold(*raw.FullName, repo.Owner+"/"+repo.Name) || raw.Private == nil || *raw.Private || raw.Archived == nil || raw.Owner == nil || raw.Owner.ID == nil || *raw.Owner.ID <= 0 || raw.Owner.Type == nil || *raw.Owner.Type != "User" || raw.Owner.Login == nil || !strings.EqualFold(*raw.Owner.Login, repo.Owner) {
		return settings.AutolinkTarget{}, fmt.Errorf("autolinks require a complete personal public repository identity")
	}
	return settings.AutolinkTarget{ID: *raw.ID, OwnerID: *raw.Owner.ID, FullName: *raw.FullName, Archived: *raw.Archived}, nil
}
func decodeAutolink(data json.RawMessage) (settings.AutolinkState, error) {
	var result settings.AutolinkState
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil || raw == nil {
		return result, fmt.Errorf("invalid autolink object")
	}
	for key := range raw {
		if key != "id" && key != "key_prefix" && key != "url_template" && key != "is_alphanumeric" && key != "updated_at" {
			return result, fmt.Errorf("unknown autolink member")
		}
	}
	var id *int64
	var prefix, target *string
	var alpha *bool
	if json.Unmarshal(raw["id"], &id) != nil || id == nil || *id <= 0 || json.Unmarshal(raw["key_prefix"], &prefix) != nil || prefix == nil || json.Unmarshal(raw["url_template"], &target) != nil || target == nil || json.Unmarshal(raw["is_alphanumeric"], &alpha) != nil || alpha == nil {
		return result, fmt.Errorf("incomplete autolink configuration")
	}
	format := "numeric"
	if *alpha {
		format = "alphanumeric"
	}
	result.Value = settings.Autolink{ID: id, ReferencePrefix: *prefix, TargetURL: *target, ReferenceFormat: format}
	if stamp, ok := raw["updated_at"]; ok {
		result.UpdatedAtPresent = true
		if json.Unmarshal(stamp, &result.UpdatedAt) != nil {
			return result, fmt.Errorf("invalid autolink timestamp")
		}
	}
	check := settings.AutolinkObservation{Target: settings.AutolinkTarget{ID: 1, OwnerID: 1, FullName: "owner/repo"}, Entries: []settings.AutolinkState{result}}
	return result, check.Validate()
}
func autolinksPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/autolinks"
}
func (c *Client) ReadAutolinks(repo repository.Repository) (settings.AutolinkObservation, error) {
	var result settings.AutolinkObservation
	var err error
	var rows []json.RawMessage
	if err = c.autolinkGET(autolinksPath(repo), &rows); err != nil {
		return result, err
	}
	if rows == nil {
		return result, fmt.Errorf("missing autolink inventory")
	}
	result.Entries = make([]settings.AutolinkState, 0, len(rows))
	for _, row := range rows {
		v, err := decodeAutolink(row)
		if err != nil {
			return result, err
		}
		result.Entries = append(result.Entries, v)
	}
	// Read the complete family first; unavailable autolinks need no extra target probe.
	result.Target, err = c.autolinkTarget(repo)
	if err != nil {
		return result, err
	}
	return result, result.Validate()
}
func (c *Client) readAutolink(repo repository.Repository, id int64) (settings.AutolinkState, error) {
	var raw json.RawMessage
	if err := c.autolinkGET(fmt.Sprintf("%s/%d", autolinksPath(repo), id), &raw); err != nil {
		return settings.AutolinkState{}, err
	}
	value, err := decodeAutolink(raw)
	if err != nil {
		return value, err
	}
	if *value.Value.ID != id {
		return value, fmt.Errorf("autolink object identity changed")
	}
	return value, nil
}
func (c *Client) CreateAutolink(repo repository.Repository, expected settings.AutolinkObservation, desired settings.Autolink) (settings.AutolinkWriteResult, error) {
	result := settings.AutolinkWriteResult{}
	current, err := c.ReadAutolinks(repo)
	if err != nil {
		return result, err
	}
	if err = settings.VerifyAutolinkInventory(expected, current); err != nil {
		return result, err
	}
	if current.Target.Archived {
		return result, fmt.Errorf("archived repository cannot create autolinks")
	}
	changes, err := settings.CompareAutolinks(current, []settings.Autolink{desired}, false)
	if err != nil {
		return result, err
	}
	if desired.ID != nil || len(changes) != 1 {
		return result, fmt.Errorf("autolink creation requires a new unbound reference")
	}
	payload, _ := json.Marshal(map[string]any{"key_prefix": desired.ReferencePrefix, "url_template": desired.TargetURL, "is_alphanumeric": desired.ReferenceFormat == "alphanumeric"})
	result.Attempted = true
	response, err := c.rest.Request(http.MethodPost, autolinksPath(repo), bytes.NewReader(payload))
	if err != nil {
		return result, fmt.Errorf("autolink creation request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != 201 {
		return result, fmt.Errorf("autolink creation acknowledgment has unexpected status")
	}
	var raw json.RawMessage
	decoder := json.NewDecoder(response.Body)
	var extra any
	if decoder.Decode(&raw) != nil {
		return result, fmt.Errorf("malformed autolink creation acknowledgment")
	}
	// Only a successful intended creation response can supply recovery evidence.
	var identity struct {
		ID *int64 `json:"id"`
	}
	if json.Unmarshal(raw, &identity) == nil && identity.ID != nil && *identity.ID > 0 {
		if _, known := expected.Find(*identity.ID); !known {
			result.RecoveryID = *identity.ID
		}
	}
	if decoder.Decode(&extra) != io.EOF || response.Header.Get("Link") != "" {
		return result, fmt.Errorf("malformed autolink creation acknowledgment")
	}
	ack, err := decodeAutolink(raw)
	if err != nil {
		return result, err
	}
	if result.RecoveryID == 0 || !settings.AutolinkMatches(ack.Value, desired) {
		return result, fmt.Errorf("autolink creation acknowledgment does not confirm a new requested reference")
	}
	actual, err := c.readAutolink(repo, result.RecoveryID)
	if err != nil {
		return result, err
	}
	if !settings.AutolinkMatches(actual.Value, desired) {
		return result, fmt.Errorf("autolink creation readback differs")
	}
	next, err := expected.WithCreated(actual)
	if err != nil {
		return result, err
	}
	inventory, err := c.ReadAutolinks(repo)
	if err != nil {
		return result, err
	}
	if err = settings.VerifyAutolinkInventory(next, inventory); err != nil {
		return result, err
	}
	result.Verified = &actual
	return result, nil
}
func (c *Client) DeleteAutolink(repo repository.Repository, expected settings.AutolinkObservation, id int64) (bool, error) {
	before, ok := expected.Find(id)
	if !ok {
		return false, fmt.Errorf("missing fixed autolink deletion candidate")
	}
	current, err := c.ReadAutolinks(repo)
	if err != nil {
		return false, err
	}
	if err = settings.VerifyAutolinkInventory(expected, current); err != nil {
		return false, err
	}
	if current.Target.Archived {
		return false, fmt.Errorf("archived repository cannot delete autolinks")
	}
	candidate, err := c.readAutolink(repo, id)
	if err != nil {
		return false, err
	}
	if !reflect.DeepEqual(before, candidate) {
		return false, fmt.Errorf("autolink deletion candidate changed")
	}
	response, err := c.rest.Request(http.MethodDelete, fmt.Sprintf("%s/%d", autolinksPath(repo), id), nil)
	if err != nil {
		return true, fmt.Errorf("autolink deletion request failed")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1))
	if err != nil || len(body) != 0 || response.StatusCode != 204 {
		return true, fmt.Errorf("autolink deletion acknowledgment is not empty 204")
	}
	actual, err := c.ReadAutolinks(repo)
	if err != nil {
		return true, err
	}
	return true, settings.VerifyAutolinkInventory(expected.Without(id), actual)
}
