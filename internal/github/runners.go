package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RunnersClient struct{ rest *api.RESTClient }

func NewRunnersClient(host string) (*RunnersClient, error) {
	if host != "github.com" {
		return nil, &securityPreWriteFailure{fmt.Errorf("runners require a github.com target")}
	}
	rest, err := api.NewRESTClient(api.ClientOptions{Host: host, LogIgnoreEnv: true, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &securityPreWriteFailure{fmt.Errorf("configure runner client")}
	}
	return &RunnersClient{rest}, nil
}

const runnerResponseLimit = 8 << 20

func runnerBody(r io.Reader) ([]byte, error) {
	b, e := io.ReadAll(io.LimitReader(r, runnerResponseLimit+1))
	if e != nil || len(b) > runnerResponseLimit {
		return nil, fmt.Errorf("runner response unavailable or oversized")
	}
	return b, nil
}
func (c *RunnersClient) get(path string, out any) (string, error) {
	r, e := c.rest.Request(http.MethodGet, path, nil)
	if e != nil {
		return "", fmt.Errorf("runner observation unavailable")
	}
	defer r.Body.Close()
	b, e := runnerBody(r.Body)
	if e != nil || r.StatusCode != 200 || json.Unmarshal(b, out) != nil {
		return "", fmt.Errorf("runner observation incomplete or unavailable")
	}
	return r.Header.Get("Link"), nil
}
func (c *RunnersClient) one(path string, out any) error {
	link, e := c.get(path, out)
	if e != nil {
		return e
	}
	if link != "" {
		return fmt.Errorf("unexpected runner pagination")
	}
	return nil
}

type runnerTarget struct {
	id, ownerID int64
	archived    bool
}

func (c *RunnersClient) target(repo repository.Repository) (runnerTarget, error) {
	var raw collaboratorRawRepository
	if repo.Host != "github.com" {
		return runnerTarget{}, fmt.Errorf("runners require a github.com target")
	}
	if e := c.one(securityRepositoryPath(repo), &raw); e != nil {
		return runnerTarget{}, e
	}
	if raw.ID == nil || *raw.ID <= 0 || raw.FullName == nil || !strings.EqualFold(*raw.FullName, repo.Owner+"/"+repo.Name) || raw.Private == nil || *raw.Private || raw.Archived == nil || raw.Owner == nil || raw.Owner.ID == nil || *raw.Owner.ID <= 0 || raw.Owner.Login == nil || !strings.EqualFold(*raw.Owner.Login, repo.Owner) || raw.Owner.Type == nil || *raw.Owner.Type != "User" {
		return runnerTarget{}, fmt.Errorf("runners require complete personal-public repository identity")
	}
	return runnerTarget{*raw.ID, *raw.Owner.ID, *raw.Archived}, nil
}

type runnerLabel struct {
	id         *int64
	name, kind string
}
type runnerRaw struct {
	ID     *int64            `json:"id"`
	Name   *string           `json:"name"`
	OS     *string           `json:"os"`
	Status *string           `json:"status"`
	Busy   *bool             `json:"busy"`
	Labels []json.RawMessage `json:"labels"`
}
type runnerState struct {
	id       int64
	name, os string
	labels   []runnerLabel
}

func decodeRunnerLabels(raw []json.RawMessage) ([]runnerLabel, error) {
	if raw == nil {
		return nil, fmt.Errorf("missing runner labels")
	}
	result := []runnerLabel{}
	names := map[string]bool{}
	ids := map[int64]bool{}
	for _, b := range raw {
		var row map[string]json.RawMessage
		var name, kind *string
		var id *int64
		if json.Unmarshal(b, &row) != nil || row == nil || json.Unmarshal(row["name"], &name) != nil || name == nil || settings.ValidateRunnerLabelName(*name) != nil || json.Unmarshal(row["type"], &kind) != nil || kind == nil || (*kind != "custom" && *kind != "read-only") {
			return nil, fmt.Errorf("runner label classification incomplete or unsupported")
		}
		if v, ok := row["id"]; ok {
			if json.Unmarshal(v, &id) != nil || id == nil || *id <= 0 || ids[*id] {
				return nil, fmt.Errorf("invalid runner label ID")
			}
			ids[*id] = true
		}
		key := settings.RunnerLabelIdentity(*name)
		if names[key] {
			return nil, fmt.Errorf("duplicate runner label identity")
		}
		names[key] = true
		result = append(result, runnerLabel{id, *name, *kind})
	}
	return result, nil
}
func decodeRunner(raw runnerRaw) (runnerState, error) {
	if raw.ID == nil || *raw.ID <= 0 || raw.Name == nil || strings.TrimSpace(*raw.Name) == "" || raw.OS == nil || strings.TrimSpace(*raw.OS) == "" || raw.Status == nil || raw.Busy == nil {
		return runnerState{}, fmt.Errorf("incomplete runner identity or state")
	}
	labels, e := decodeRunnerLabels(raw.Labels)
	return runnerState{*raw.ID, *raw.Name, *raw.OS, labels}, e
}
func decodeRunnerEnvelope(raw json.RawMessage) ([]runnerLabel, error) {
	var v struct {
		Total  *int              `json:"total_count"`
		Labels []json.RawMessage `json:"labels"`
	}
	if json.Unmarshal(raw, &v) != nil || v.Total == nil || *v.Total < 0 || *v.Total != len(v.Labels) {
		return nil, fmt.Errorf("incomplete runner label envelope")
	}
	return decodeRunnerLabels(v.Labels)
}

// Join independent reads without discarding optional ID evidence from either.
func mergeRunnerLabels(a, b []runnerLabel) ([]runnerLabel, error) {
	if len(a) != len(b) {
		return nil, fmt.Errorf("inconsistent runner label reads")
	}
	result := append([]runnerLabel{}, a...)
	for i, x := range a {
		found := false
		for _, y := range b {
			if settings.RunnerLabelIdentity(x.name) != settings.RunnerLabelIdentity(y.name) {
				continue
			}
			found = true
			if x.kind != y.kind || (x.kind == "read-only" && x.name != y.name) || (x.id != nil && y.id != nil && *x.id != *y.id) {
				return nil, fmt.Errorf("inconsistent runner label evidence")
			}
			if result[i].id == nil {
				result[i].id = y.id
			}
		}
		if !found {
			return nil, fmt.Errorf("inconsistent runner labels")
		}
	}
	// IDs supplied on opposite endpoints must not identify two different labels.
	ids := map[int64]bool{}
	for _, v := range result {
		if v.id != nil {
			if ids[*v.id] {
				return nil, fmt.Errorf("conflicting runner label IDs")
			}
			ids[*v.id] = true
		}
	}
	return result, nil
}
func runnerPath(repo repository.Repository) string {
	return securityRepositoryPath(repo) + "/actions/runners"
}
func (c *RunnersClient) readRunner(repo repository.Repository, id int64) (runnerState, error) {
	var raw runnerRaw
	path := runnerPath(repo) + "/" + strconv.FormatInt(id, 10)
	if e := c.one(path, &raw); e != nil {
		return runnerState{}, e
	}
	v, e := decodeRunner(raw)
	if e != nil {
		return v, e
	}
	if v.id != id {
		return v, fmt.Errorf("runner identity changed")
	}
	var envelope json.RawMessage
	if e = c.one(path+"/labels", &envelope); e != nil {
		return v, e
	}
	labels, e := decodeRunnerEnvelope(envelope)
	if e != nil {
		return v, e
	}
	v.labels, e = mergeRunnerLabels(v.labels, labels)
	return v, e
}
func runnerProjection(v runnerState) settings.RunnerState {
	custom, readonly := []string{}, []string{}
	for _, l := range v.labels {
		if l.kind == "custom" {
			custom = append(custom, l.name)
		} else {
			readonly = append(readonly, l.name)
		}
	}
	return settings.RunnerState{Value: settings.RunnerLabels{ID: v.id, CustomLabels: &custom}, ReadOnlyLabels: readonly}
}
func (c *RunnersClient) ReadSelected(repo repository.Repository, desired []settings.RunnerLabels) (settings.RunnerObservation, error) {
	if e := settings.ValidateRunnerLabels(desired); e != nil {
		return settings.RunnerObservation{}, e
	}
	entries := []settings.RunnerState{}
	if len(desired) == 0 {
		return settings.NewRunnerObservation(entries)
	}
	if _, e := c.target(repo); e != nil {
		return settings.RunnerObservation{}, e
	}
	for _, d := range desired {
		v, e := c.readRunner(repo, d.ID)
		if e != nil {
			return settings.RunnerObservation{}, e
		}
		entries = append(entries, runnerProjection(v))
	}
	return settings.NewRunnerObservation(entries)
}

// Pagination requests are constructed locally; Link never supplies a request target.
func runnerNextPage(link string, repo repository.Repository, page int, more bool) error {
	next := false
	for _, part := range strings.Split(link, ",") {
		if !strings.Contains(part, `rel="next"`) {
			continue
		}
		if next {
			return fmt.Errorf("duplicate runner next page")
		}
		next = true
		start, end := strings.Index(part, "<"), strings.Index(part, ">")
		if start < 0 || end <= start {
			return fmt.Errorf("invalid runner pagination")
		}
		u, e := url.Parse(part[start+1 : end])
		if e != nil || u.Scheme != "https" || u.Host != "api.github.com" || u.User != nil || u.Fragment != "" || u.Path != "/"+strings.TrimPrefix(runnerPath(repo), "/") {
			return fmt.Errorf("unsafe runner pagination")
		}
		q, e := url.ParseQuery(u.RawQuery)
		if e != nil || len(q) != 2 || len(q["page"]) != 1 || q.Get("page") != strconv.Itoa(page+1) || len(q["per_page"]) != 1 || q.Get("per_page") != "100" {
			return fmt.Errorf("unexpected runner pagination query")
		}
	}
	if next != more {
		return fmt.Errorf("incomplete runner pagination")
	}
	return nil
}
func (c *RunnersClient) ReadAll(repo repository.Repository) (settings.RunnerObservation, error) {
	_, e := c.target(repo)
	if e != nil {
		return settings.RunnerObservation{}, e
	}
	ids := []int64{}
	seen := map[int64]bool{}
	total := -1
	for page := 1; page <= 10000; page++ {
		var raw struct {
			Total   *int        `json:"total_count"`
			Runners []runnerRaw `json:"runners"`
		}
		link, e := c.get(fmt.Sprintf("%s?per_page=100&page=%d", runnerPath(repo), page), &raw)
		if e != nil {
			return settings.RunnerObservation{}, e
		}
		if raw.Total == nil || *raw.Total < 0 || raw.Runners == nil || len(raw.Runners) > 100 || (total >= 0 && total != *raw.Total) {
			return settings.RunnerObservation{}, fmt.Errorf("incomplete runner inventory")
		}
		total = *raw.Total
		for _, r := range raw.Runners {
			v, e := decodeRunner(r)
			if e != nil {
				return settings.RunnerObservation{}, e
			}
			if seen[v.id] {
				return settings.RunnerObservation{}, fmt.Errorf("duplicate runner inventory identity")
			}
			seen[v.id] = true
			ids = append(ids, v.id)
		}
		if len(ids) > total {
			return settings.RunnerObservation{}, fmt.Errorf("runner total inconsistent")
		}
		more := len(ids) < total
		if (more && len(raw.Runners) != 100) || runnerNextPage(link, repo, page, more) != nil {
			return settings.RunnerObservation{}, fmt.Errorf("runner pagination incomplete or unsafe")
		}
		if !more {
			break
		}
		if page == 10000 {
			return settings.RunnerObservation{}, fmt.Errorf("runner inventory too large")
		}
	}
	entries := []settings.RunnerState{}
	for _, id := range ids {
		v, e := c.readRunner(repo, id)
		if e != nil {
			return settings.RunnerObservation{}, e
		}
		entries = append(entries, runnerProjection(v))
	}
	return settings.NewRunnerObservation(entries)
}
func runnerLabelsVerified(before []runnerLabel, actual []runnerLabel, desired settings.RunnerLabels) bool {
	custom := []string{}
	old := map[string]runnerLabel{}
	for _, v := range before {
		if v.kind == "read-only" {
			old[v.name] = v
		}
	}
	count := 0
	for _, v := range actual {
		if v.kind == "custom" {
			custom = append(custom, v.name)
			continue
		}
		count++
		prior, ok := old[v.name]
		if !ok || (prior.id != nil && (v.id == nil || *v.id != *prior.id)) {
			return false
		}
	}
	return count == len(old) && settings.RunnerLabelsMatch(settings.RunnerLabels{ID: desired.ID, CustomLabels: &custom}, desired)
}
func (c *RunnersClient) Update(repo repository.Repository, desired settings.RunnerLabels) (bool, error) {
	pre := func(e error) (bool, error) { return false, &securityPreWriteFailure{e} }
	if e := settings.ValidateRunnerLabels([]settings.RunnerLabels{desired}); e != nil {
		return pre(e)
	}
	desired = settings.CloneRunnerLabels([]settings.RunnerLabels{desired})[0]
	target, e := c.target(repo)
	if e != nil {
		return pre(e)
	}
	before, e := c.readRunner(repo, desired.ID)
	if e != nil {
		return pre(e)
	}
	observation, e := settings.NewRunnerObservation([]settings.RunnerState{runnerProjection(before)})
	if e != nil {
		return pre(e)
	}
	if e = settings.PrepareRunnerLabels(observation, []settings.RunnerLabels{desired}); e != nil {
		return pre(e)
	}
	if settings.RunnerLabelsMatch(runnerProjection(before).Value, desired) {
		return false, nil
	}
	if target.archived {
		return pre(fmt.Errorf("archived repository cannot update runner labels"))
	}
	b, _ := json.Marshal(map[string]any{"labels": *desired.CustomLabels})
	r, e := c.rest.Request(http.MethodPut, runnerPath(repo)+"/"+strconv.FormatInt(desired.ID, 10)+"/labels", bytes.NewReader(b))
	if e != nil {
		return false, fmt.Errorf("runner labels update failed; effects unknown")
	}
	raw, e := runnerBody(r.Body)
	r.Body.Close()
	if e != nil || r.StatusCode != 200 || r.Header.Get("Link") != "" {
		return false, fmt.Errorf("runner labels acknowledgment unavailable; effects unknown")
	}
	ack, e := decodeRunnerEnvelope(raw)
	if e != nil || !runnerLabelsVerified(before.labels, ack, desired) {
		return false, fmt.Errorf("runner labels acknowledgment did not confirm requested labels and preservation; effects unknown")
	}
	afterTarget, e := c.target(repo)
	if e != nil || afterTarget != target {
		return false, fmt.Errorf("runner repository readback differs or unavailable; effects unknown")
	}
	after, e := c.readRunner(repo, desired.ID)
	if e != nil || after.name != before.name || after.os != before.os || !runnerLabelsVerified(before.labels, after.labels, desired) {
		return false, fmt.Errorf("runner readback did not confirm requested labels and preserved identity; effects unknown")
	}
	return true, nil
}
