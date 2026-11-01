package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type PagesClient struct {
	rest                 *api.RESTClient
	NormalizationWarning func(settings.ObservationNormalization)
}
type pagesPreWriteFailure struct{ cause error }

func (e *pagesPreWriteFailure) Error() string        { return e.cause.Error() }
func (e *pagesPreWriteFailure) Unwrap() error        { return e.cause }
func (*pagesPreWriteFailure) ReadFailedBeforeWrite() {}
func NewPagesClient(host string) (*PagesClient, error) {
	c, err := api.NewRESTClient(api.ClientOptions{Host: host, Timeout: 30 * time.Second, Headers: map[string]string{"X-GitHub-Api-Version": "2026-03-10"}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		return nil, &pagesPreWriteFailure{fmt.Errorf("configure Pages API: %w", err)}
	}
	return &PagesClient{rest: c}, nil
}
func pagesPath(repo repository.Repository) string {
	return "repos/" + repo.Owner + "/" + repo.Name + "/pages"
}
func (c *PagesClient) Read(repo repository.Repository) (settings.PagesObservation, error) {
	var raw map[string]json.RawMessage
	response, err := c.rest.Request(http.MethodGet, pagesPath(repo), nil)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
			return settings.NewAbsentPagesObservation(), nil
		}
		return settings.PagesObservation{}, fmt.Errorf("read Pages: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return settings.PagesObservation{}, fmt.Errorf("unexpected Pages read status %d", response.StatusCode)
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&raw); err != nil {
		return settings.PagesObservation{}, fmt.Errorf("decode Pages: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return settings.PagesObservation{}, fmt.Errorf("GitHub returned trailing or incomplete Pages JSON")
	}
	var mode *string
	var https, public *bool
	var cname *string
	if json.Unmarshal(raw["build_type"], &mode) != nil || mode == nil || json.Unmarshal(raw["https_enforced"], &https) != nil || https == nil || json.Unmarshal(raw["public"], &public) != nil || public == nil || raw["cname"] == nil || json.Unmarshal(raw["cname"], &cname) != nil {
		return settings.PagesObservation{}, fmt.Errorf("GitHub returned incomplete or invalid Pages configured state")
	}
	preparedMode, err := preparePagesModeValue(mode)
	if err != nil {
		return settings.PagesObservation{}, err
	}
	s := settings.PagesState{BuildType: preparedMode.BuildType, CNAME: cname, HTTPSEnforced: *https, Public: *public}
	var record settings.ObservationNormalization
	if s.BuildType == "legacy" {
		var source map[string]json.RawMessage
		var branch, path *string
		if json.Unmarshal(raw["source"], &source) != nil || json.Unmarshal(source["branch"], &branch) != nil || branch == nil || json.Unmarshal(source["path"], &path) != nil || path == nil {
			return settings.PagesObservation{}, fmt.Errorf("GitHub returned incomplete legacy Pages source")
		}
		prepared, err := prepareLegacyPagesFolderObservation(branch, path)
		if err != nil {
			return settings.PagesObservation{}, err
		}
		s.Source, record = &prepared.Source, prepared.Normalization
	}
	if preparedMode.Normalization != (settings.ObservationNormalization{}) {
		return settings.NewPagesObservationWithModeNormalization(s, preparedMode.Normalization, record)
	}
	if record != (settings.ObservationNormalization{}) {
		return settings.NewPagesObservationWithFolderNormalization(s, record)
	}
	return settings.NewPagesObservation(s)
}

// Update reconciles the declared publishing controls, including explicit creation/deletion.
func (c *PagesClient) Update(repo repository.Repository, d settings.PagesDeclaration) (bool, error) {
	return c.reconcile(repo, d, nil)
}

func (c *PagesClient) reconcile(repo repository.Repository, d settings.PagesDeclaration, requiredIdentity *settings.ArchiveState) (bool, error) {
	pre := func(err error) (bool, error) { return false, &pagesPreWriteFailure{err} }
	if err := d.Validate(); err != nil {
		return pre(err)
	}
	if d.Empty() {
		return false, nil
	}
	identity := &Client{rest: c.rest}
	initial, err := identity.ReadArchive(repo)
	if err != nil {
		return pre(err)
	}
	if requiredIdentity != nil && initial != *requiredIdentity {
		return pre(fmt.Errorf("repository identity changed before Pages follow-up"))
	}
	if initial.Archived {
		return pre(fmt.Errorf("archived repository cannot update Pages without explicit unarchive"))
	}
	observed, err := c.Read(repo)
	if err != nil {
		return pre(err)
	}
	c.warnPagesNormalizations(observed, d)
	_, changes, err := settings.ComparePages(observed, d)
	if err != nil {
		return pre(err)
	}
	if len(changes) == 0 {
		return false, nil
	}
	if d.Delete() {
		return c.deleteSite(repo, initial)
	}
	if absent, _ := observed.Absent(); absent {
		return c.createSite(repo, initial, d)
	}
	expected, err := preparePagesVerificationState(observed, d)
	if err != nil {
		return pre(err)
	}
	payload := map[string]any{}
	if d.BuildType != nil {
		payload["build_type"] = *d.BuildType
	}
	if d.Source != nil {
		payload["source"] = *d.Source
	}
	if d.CNAME != nil {
		payload["cname"] = expected.CNAME
	}
	if d.HTTPSEnforced != nil {
		payload["https_enforced"] = *d.HTTPSEnforced
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return pre(err)
	}
	fresh, err := identity.ReadArchive(repo)
	if err != nil {
		return pre(err)
	}
	if fresh != initial {
		return pre(fmt.Errorf("repository identity or archive state changed before Pages update"))
	}
	response, err := c.rest.Request(http.MethodPut, pagesPath(repo), bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("update Pages (write may have applied): %w", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return false, fmt.Errorf("unexpected Pages update status %d (write may have applied)", response.StatusCode)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify Pages (write may have applied): %w", err)
	}
	c.warnPagesNormalizations(actual, d)
	retained, err := samePagesVerificationState(expected, actual)
	if err != nil || !retained {
		return false, fmt.Errorf("Pages readback did not retain requested values, omitted companions or visibility (write may have applied)")
	}
	final, err := identity.ReadArchive(repo)
	if err != nil || final != initial {
		return false, fmt.Errorf("Pages readback did not confirm repository identity and archive state (write may have applied)")
	}
	return true, nil
}
