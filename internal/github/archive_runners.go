package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// archivedRunnerRead preserves optional IDs separately for each endpoint.
type archivedRunnerRead struct {
	state        runnerState
	detail, list []runnerLabel
}

func (c *RunnersClient) archivedJSON(path string, out any) error {
	response, err := c.rest.Request(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("runner observation unavailable")
	}
	defer response.Body.Close()
	raw, err := runnerBody(response.Body)
	if err != nil || response.StatusCode != http.StatusOK || len(response.Header.Values("Link")) != 0 || !validVariableJSONUnicode(raw) || runnerPruneJSON(raw, out) != nil {
		return fmt.Errorf("runner observation incomplete or ambiguous")
	}
	return nil
}
func (c *RunnersClient) archivedTarget(repo repository.Repository, archive settings.ArchiveState) (runnerTarget, error) {
	var fields map[string]json.RawMessage
	if repo.Host != "github.com" || !archive.Archived || archive.Validate(repo) != nil || c.archivedJSON(securityRepositoryPath(repo), &fields) != nil || fields == nil || deployKeyCaseAlias(fields, "id", "full_name", "private", "visibility", "archived", "owner") {
		return runnerTarget{}, fmt.Errorf("runner target identity unavailable")
	}
	var owner map[string]json.RawMessage
	if json.Unmarshal(fields["owner"], &owner) != nil || owner == nil || deployKeyCaseAlias(owner, "id", "login", "type") {
		return runnerTarget{}, fmt.Errorf("runner owner identity unavailable")
	}
	var raw collaboratorRawRepository
	data, _ := json.Marshal(fields)
	if json.Unmarshal(data, &raw) != nil || raw.ID == nil || *raw.ID <= 0 || *raw.ID != archive.ID || raw.FullName == nil || !strings.EqualFold(*raw.FullName, archive.FullName) || !strings.EqualFold(*raw.FullName, repo.Owner+"/"+repo.Name) || raw.Private == nil || *raw.Private || raw.Archived == nil || !*raw.Archived || raw.Owner == nil || raw.Owner.ID == nil || *raw.Owner.ID <= 0 || raw.Owner.Login == nil || !strings.EqualFold(*raw.Owner.Login, repo.Owner) || raw.Owner.Type == nil || *raw.Owner.Type != "User" {
		return runnerTarget{}, fmt.Errorf("runner target identity unavailable")
	}
	if visibility, ok := fields["visibility"]; ok {
		var v string
		if json.Unmarshal(visibility, &v) != nil || v != "public" {
			return runnerTarget{}, fmt.Errorf("runner target visibility unavailable")
		}
	}
	return runnerTarget{*raw.ID, *raw.Owner.ID, *raw.Archived}, nil
}
func strictArchivedLabels(raw []json.RawMessage) ([]runnerLabel, error) {
	for _, item := range raw {
		var row map[string]json.RawMessage
		if runnerPruneJSON(item, &row) != nil || row == nil || deployKeyCaseAlias(row, "id", "name", "type") {
			return nil, fmt.Errorf("ambiguous runner label")
		}
		for key := range row {
			if key != "id" && key != "name" && key != "type" {
				return nil, fmt.Errorf("unsupported runner label")
			}
		}
	}
	return decodeRunnerLabels(raw)
}
func (c *RunnersClient) archivedRunner(repo repository.Repository, id int64) (archivedRunnerRead, error) {
	path := runnerPath(repo) + "/" + strconv.FormatInt(id, 10)
	var detail map[string]json.RawMessage
	if c.archivedJSON(path, &detail) != nil || detail == nil || deployKeyCaseAlias(detail, "id", "name", "os", "status", "busy", "labels") {
		return archivedRunnerRead{}, fmt.Errorf("runner detail incomplete or ambiguous")
	}
	var raw runnerRaw
	data, _ := json.Marshal(detail)
	if json.Unmarshal(data, &raw) != nil {
		return archivedRunnerRead{}, fmt.Errorf("runner detail incomplete")
	}
	state, err := decodeRunner(raw)
	if err != nil || state.id != id {
		return archivedRunnerRead{}, fmt.Errorf("runner detail identity incomplete")
	}
	labels, err := strictArchivedLabels(raw.Labels)
	if err != nil {
		return archivedRunnerRead{}, err
	}
	var envelope map[string]json.RawMessage
	if c.archivedJSON(path+"/labels", &envelope) != nil || envelope == nil || deployKeyCaseAlias(envelope, "total_count", "labels") {
		return archivedRunnerRead{}, fmt.Errorf("runner label list incomplete or ambiguous")
	}
	var count *int
	var rows []json.RawMessage
	if json.Unmarshal(envelope["total_count"], &count) != nil || count == nil || *count < 0 || json.Unmarshal(envelope["labels"], &rows) != nil || rows == nil || *count != len(rows) {
		return archivedRunnerRead{}, fmt.Errorf("runner label list incomplete")
	}
	listed, err := strictArchivedLabels(rows)
	if err != nil {
		return archivedRunnerRead{}, err
	}
	if _, err = mergeRunnerLabels(labels, listed); err != nil {
		return archivedRunnerRead{}, err
	}
	return archivedRunnerRead{state: state, detail: labels, list: listed}, nil
}
func sameArchivedRunnerLabels(a, b []runnerLabel) bool {
	if len(a) != len(b) {
		return false
	}
	for _, x := range a {
		found := false
		for _, y := range b {
			if settings.RunnerLabelIdentity(x.name) != settings.RunnerLabelIdentity(y.name) {
				continue
			}
			if x.kind != y.kind || x.kind == "read-only" && x.name != y.name || (x.id == nil) != (y.id == nil) || x.id != nil && *x.id != *y.id {
				return false
			}
			found = true
			break
		}
		if !found {
			return false
		}
	}
	return true
}

// VerifyArchivedMatching performs independent observations; they are not atomic.
func (c *RunnersClient) VerifyArchivedMatching(repo repository.Repository, desired []settings.RunnerLabels, archive settings.ArchiveState) error {
	if err := settings.ValidateRunnerLabels(desired); err != nil {
		return err
	}
	if len(desired) == 0 {
		return nil
	}
	var firstTarget runnerTarget
	first := map[int64]archivedRunnerRead{}
	for pass := 0; pass < 2; pass++ {
		target, err := c.archivedTarget(repo, archive)
		if err != nil {
			return err
		}
		if pass == 0 {
			firstTarget = target
		} else if target != firstTarget {
			return fmt.Errorf("runner target identity changed")
		}
		for _, d := range desired {
			current, err := c.archivedRunner(repo, d.ID)
			if err != nil {
				return err
			}
			observed, err := settings.NewRunnerObservation([]settings.RunnerState{runnerProjection(current.state)})
			if err != nil {
				return err
			}
			_, changes, err := settings.CompareRunnerLabels(observed, []settings.RunnerLabels{d})
			if err != nil || len(changes) != 0 {
				return fmt.Errorf("runner custom labels differ or are incomplete")
			}
			if pass == 0 {
				first[d.ID] = current
				continue
			}
			before := first[d.ID]
			if before.state.id != current.state.id || before.state.name != current.state.name || before.state.os != current.state.os || !sameArchivedRunnerLabels(before.detail, current.detail) || !sameArchivedRunnerLabels(before.list, current.list) {
				return fmt.Errorf("runner identity or labels changed")
			}
		}
	}
	return nil
}
