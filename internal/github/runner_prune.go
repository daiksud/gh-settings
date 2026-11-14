package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Reject duplicate keys recursively before decoding evidence.
func runnerPruneJSON(data []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var walk func() error
	walk = func() error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return fmt.Errorf("duplicate runner member")
				}
				seen[name] = true
				if err = walk(); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := walk(); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("invalid runner response")
		}
		_, err = d.Token()
		return err
	}
	if walk() != nil {
		return fmt.Errorf("invalid runner response")
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("invalid runner response")
	}
	return json.Unmarshal(data, out)
}
func (c *RunnersClient) pruneTarget(repo repository.Repository) (runnerTarget, error) {
	if repo.Host != "github.com" {
		return runnerTarget{}, fmt.Errorf("runners require a github.com target")
	}
	var data json.RawMessage
	if err := c.one(securityRepositoryPath(repo), &data); err != nil {
		return runnerTarget{}, err
	}
	var raw collaboratorRawRepository
	if runnerPruneJSON(data, &raw) != nil || raw.ID == nil || *raw.ID <= 0 || raw.FullName == nil || !strings.EqualFold(*raw.FullName, repo.Owner+"/"+repo.Name) || raw.Private == nil || *raw.Private || raw.Archived == nil || raw.Owner == nil || raw.Owner.ID == nil || *raw.Owner.ID <= 0 || raw.Owner.Login == nil || !strings.EqualFold(*raw.Owner.Login, repo.Owner) || raw.Owner.Type == nil || *raw.Owner.Type != "User" {
		return runnerTarget{}, fmt.Errorf("runners require complete personal-public repository identity")
	}
	return runnerTarget{*raw.ID, *raw.Owner.ID, *raw.Archived}, nil
}
func pruneRunnerLabels(raw []json.RawMessage) ([]settings.RunnerPruneLabel, error) {
	for _, b := range raw {
		var row map[string]json.RawMessage
		if runnerPruneJSON(b, &row) != nil || row == nil {
			return nil, fmt.Errorf("invalid runner label")
		}
		for k := range row {
			if k != "id" && k != "name" && k != "type" {
				return nil, fmt.Errorf("unknown runner label member")
			}
		}
	}
	ls, err := decodeRunnerLabels(raw)
	if err != nil {
		return nil, err
	}
	out := []settings.RunnerPruneLabel{}
	for _, l := range ls {
		out = append(out, settings.RunnerPruneLabel{ID: l.id, Name: l.name, Kind: l.kind})
	}
	slices.SortFunc(out, func(a, b settings.RunnerPruneLabel) int {
		return strings.Compare(settings.RunnerLabelIdentity(a.Name), settings.RunnerLabelIdentity(b.Name))
	})
	return out, nil
}
func pruneRunner(raw json.RawMessage) (settings.RunnerPruneState, error) {
	var fields map[string]json.RawMessage
	if runnerPruneJSON(raw, &fields) != nil || fields == nil {
		return settings.RunnerPruneState{}, fmt.Errorf("invalid runner")
	}
	for k := range fields {
		switch k {
		case "id", "name", "os", "status", "busy", "labels", "ephemeral", "version":
		default:
			return settings.RunnerPruneState{}, fmt.Errorf("unknown runner member")
		}
	}
	var v struct {
		ID     *int64            `json:"id"`
		Name   *string           `json:"name"`
		OS     *string           `json:"os"`
		Status *string           `json:"status"`
		Busy   *bool             `json:"busy"`
		Labels []json.RawMessage `json:"labels"`
	}
	if json.Unmarshal(raw, &v) != nil || v.ID == nil || v.Name == nil || v.OS == nil || v.Status == nil || v.Busy == nil {
		return settings.RunnerPruneState{}, fmt.Errorf("incomplete runner state")
	}
	result := settings.RunnerPruneState{ID: *v.ID, Name: *v.Name, OS: *v.OS, Status: *v.Status, Busy: *v.Busy}
	if result.ID <= 0 || strings.TrimSpace(result.Name) == "" || strings.TrimSpace(result.OS) == "" || (result.Status != "online" && result.Status != "offline") {
		return result, fmt.Errorf("invalid runner identity or state")
	}
	if b, ok := fields["ephemeral"]; ok {
		var x *bool
		if json.Unmarshal(b, &x) != nil || x == nil {
			return result, fmt.Errorf("invalid runner ephemeral evidence")
		}
		result.Ephemeral = x
	}
	if b, ok := fields["version"]; ok {
		var x *string
		if json.Unmarshal(b, &x) != nil || x == nil {
			return result, fmt.Errorf("invalid runner version evidence")
		}
		if strings.TrimSpace(*x) == "" {
			return result, fmt.Errorf("invalid runner version evidence")
		}
		result.Version = x
	}
	var err error
	result.Labels, err = pruneRunnerLabels(v.Labels)
	return result, err
}
func pruneLabelsEnvelope(raw json.RawMessage) ([]settings.RunnerPruneLabel, error) {
	var fields map[string]json.RawMessage
	if runnerPruneJSON(raw, &fields) != nil || len(fields) != 2 {
		return nil, fmt.Errorf("invalid runner labels envelope")
	}
	for k := range fields {
		if k != "total_count" && k != "labels" {
			return nil, fmt.Errorf("unknown runner labels member")
		}
	}
	var v struct {
		Total  *int              `json:"total_count"`
		Labels []json.RawMessage `json:"labels"`
	}
	if json.Unmarshal(raw, &v) != nil || v.Total == nil || v.Labels == nil || *v.Total != len(v.Labels) {
		return nil, fmt.Errorf("incomplete runner labels")
	}
	return pruneRunnerLabels(v.Labels)
}
func compatiblePruneLabels(a, b []settings.RunnerPruneLabel) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		y := b[i]
		if x.Name != y.Name || x.Kind != y.Kind || (x.ID != nil && y.ID != nil && *x.ID != *y.ID) {
			return false
		}
	}
	return true
}
func mergePruneLabels(groups ...[]settings.RunnerPruneLabel) ([]settings.RunnerPruneLabel, error) {
	if len(groups) == 0 {
		return nil, fmt.Errorf("missing runner labels")
	}
	merged := append([]settings.RunnerPruneLabel{}, groups[0]...)
	for _, group := range groups[1:] {
		if !compatiblePruneLabels(merged, group) {
			return nil, fmt.Errorf("inconsistent runner labels")
		}
		for i, l := range group {
			if merged[i].ID == nil && l.ID != nil {
				merged[i].ID = l.ID
			}
		}
	}
	ids := map[int64]bool{}
	for _, l := range merged {
		if l.ID != nil {
			if ids[*l.ID] {
				return nil, fmt.Errorf("conflicting runner label IDs")
			}
			ids[*l.ID] = true
		}
	}
	return merged, nil
}
func (c *RunnersClient) pruneList(repo repository.Repository) ([]settings.RunnerPruneState, error) {
	out := []settings.RunnerPruneState{}
	seen := map[int64]bool{}
	total := -1
	for page := 1; page <= 10000; page++ {
		var envelope json.RawMessage
		link, e := c.get(fmt.Sprintf("%s?per_page=100&page=%d", runnerPath(repo), page), &envelope)
		var members map[string]json.RawMessage
		if runnerPruneJSON(envelope, &members) != nil || len(members) != 2 || members["total_count"] == nil || members["runners"] == nil {
			return nil, fmt.Errorf("unknown or incomplete runner inventory envelope")
		}
		var raw struct {
			Total   *int              `json:"total_count"`
			Runners []json.RawMessage `json:"runners"`
		}
		if json.Unmarshal(envelope, &raw) != nil {
			return nil, fmt.Errorf("invalid runner inventory envelope")
		}
		if e != nil {
			return nil, e
		}
		if raw.Total == nil || *raw.Total < 0 || raw.Runners == nil || len(raw.Runners) > 100 || (total >= 0 && total != *raw.Total) {
			return nil, fmt.Errorf("incomplete runner inventory")
		}
		total = *raw.Total
		for _, row := range raw.Runners {
			v, e := pruneRunner(row)
			if e != nil {
				return nil, e
			}
			if seen[v.ID] {
				return nil, fmt.Errorf("duplicate runner identity")
			}
			seen[v.ID] = true
			out = append(out, v)
		}
		if len(out) > total {
			return nil, fmt.Errorf("inconsistent runner count")
		}
		more := len(out) < total
		if (more && len(raw.Runners) != 100) || runnerNextPage(link, repo, page, more) != nil {
			return nil, fmt.Errorf("incomplete runner pagination")
		}
		if !more {
			slices.SortFunc(out, func(a, b settings.RunnerPruneState) int {
				if a.ID < b.ID {
					return -1
				}
				if a.ID > b.ID {
					return 1
				}
				return 0
			})
			return out, nil
		}
	}
	return nil, fmt.Errorf("runner inventory too large")
}
func (c *RunnersClient) ReadPrune(repo repository.Repository) (settings.RunnerPruneObservation, error) {
	target, e := c.pruneTarget(repo)
	if e != nil {
		return settings.RunnerPruneObservation{}, e
	}
	list, e := c.pruneList(repo)
	if e != nil {
		return settings.RunnerPruneObservation{}, e
	}
	out := settings.RunnerPruneObservation{Target: settings.RunnerPruneTarget{ID: target.id, OwnerID: target.ownerID, Archived: target.archived}, Entries: []settings.RunnerPruneState{}}
	for _, v := range list {
		path := runnerPath(repo) + "/" + strconv.FormatInt(v.ID, 10)
		var raw json.RawMessage
		if e = c.one(path, &raw); e != nil {
			return out, e
		}
		detail, e := pruneRunner(raw)
		if e != nil {
			return out, e
		}
		var env json.RawMessage
		if e = c.one(path+"/labels", &env); e != nil {
			return out, e
		}
		labels, e := pruneLabelsEnvelope(env)
		if e != nil {
			return out, e
		}
		if v.ID != detail.ID || v.Name != detail.Name || v.OS != detail.OS || !reflect.DeepEqual(v.Ephemeral, detail.Ephemeral) || !compatiblePruneLabels(v.Labels, detail.Labels) || !compatiblePruneLabels(detail.Labels, labels) || !compatiblePruneLabels(v.Labels, labels) {
			return out, fmt.Errorf("inconsistent runner observations")
		}
		merged, e := mergePruneLabels(v.Labels, detail.Labels, labels)
		if e != nil {
			return out, e
		}
		detail.HasOperationalReads = true
		detail.ListStatus = v.Status
		detail.ListBusy = v.Busy
		detail.ListVersion = v.Version
		detail.ListLabels = v.Labels
		detail.DetailLabels = detail.Labels
		detail.EndpointLabels = labels
		detail.Labels = merged
		out.Entries = append(out.Entries, detail)
	}
	end, e := c.pruneList(repo)
	if e != nil || len(list) != len(end) {
		return out, fmt.Errorf("runner inventory changed during acquisition")
	}
	for i, first := range list {
		last := end[i]
		first.Status = ""
		first.Busy = false
		first.Version = nil
		last.Status = ""
		last.Busy = false
		last.Version = nil
		if !reflect.DeepEqual(first, last) {
			return out, fmt.Errorf("runner inventory changed during acquisition")
		}
		out.Entries[i].LastStatus = end[i].Status
		out.Entries[i].LastBusy = end[i].Busy
		out.Entries[i].LastVersion = end[i].Version
	}
	target2, e := c.pruneTarget(repo)
	if e != nil || target != target2 {
		return out, fmt.Errorf("runner repository changed during acquisition")
	}
	if e = out.Validate(nil); e != nil {
		return out, e
	}
	return out, nil
}
func (c *RunnersClient) DeletePrune(repo repository.Repository, expected settings.RunnerPruneObservation, candidates []int64, id int64) (bool, error) {
	before, e := c.ReadPrune(repo)
	if e != nil {
		return false, &securityPreWriteFailure{e}
	}
	if e = settings.VerifyRunnerPrune(expected, before, candidates, nil); e != nil {
		return false, &securityPreWriteFailure{e}
	}
	if before.Target.Archived {
		return false, &securityPreWriteFailure{fmt.Errorf("archived repository cannot delete runners")}
	}
	found := false
	for _, v := range candidates {
		if v == id {
			found = true
		}
	}
	if !found {
		return false, &securityPreWriteFailure{fmt.Errorf("runner is not a fixed candidate")}
	}
	r, e := c.rest.Request(http.MethodDelete, runnerPath(repo)+"/"+strconv.FormatInt(id, 10), nil)
	if e != nil {
		return true, fmt.Errorf("runner deletion attempted; effects unknown")
	}
	body, e := runnerBody(r.Body)
	r.Body.Close()
	if e != nil || r.StatusCode != 204 || len(body) != 0 || r.Header.Get("Link") != "" {
		return true, fmt.Errorf("runner deletion acknowledgment invalid; effects unknown")
	}
	want, e := expected.Without(id)
	if e != nil {
		return true, fmt.Errorf("runner deletion attempted; effects unknown")
	}
	remaining := []int64{}
	for _, v := range candidates {
		if v != id {
			remaining = append(remaining, v)
		}
	}
	after, e := c.ReadPrune(repo)
	if e != nil || settings.VerifyRunnerPrune(want, after, remaining, nil) != nil {
		return true, fmt.Errorf("runner deletion could not be independently confirmed; effects unknown")
	}
	return true, nil
}
