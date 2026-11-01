package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// A confirmed creation remains progress even if its later optional update fails.
type pagesCreatedFailure struct{ cause error }

func (e *pagesCreatedFailure) Error() string {
	return "Pages site created; follow-up configuration failed (creation remains applied): " + e.cause.Error()
}
func (e *pagesCreatedFailure) Unwrap() error         { return e.cause }
func (*pagesCreatedFailure) PagesCreationCompleted() {}

func (c *PagesClient) checkLifecycleIdentity(repo repository.Repository, initial settings.ArchiveState) error {
	fresh, err := (&Client{rest: c.rest}).ReadArchive(repo)
	if err != nil {
		return err
	}
	if fresh != initial {
		return fmt.Errorf("repository identity or archive state changed during Pages lifecycle operation")
	}
	return nil
}
func (c *PagesClient) deleteSite(repo repository.Repository, initial settings.ArchiveState) (bool, error) {
	if err := c.checkLifecycleIdentity(repo, initial); err != nil {
		return false, &pagesPreWriteFailure{err}
	}
	response, err := c.rest.Request(http.MethodDelete, pagesPath(repo), nil)
	if err != nil {
		var httpErr *api.HTTPError
		if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
			return false, fmt.Errorf("delete Pages (write may have applied): %w", err)
		}
	} else {
		response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			return false, fmt.Errorf("unexpected Pages delete status %d (write may have applied)", response.StatusCode)
		}
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify Pages deletion (write may have applied): %w", err)
	}
	absent, err := actual.Absent()
	if err != nil || !absent {
		return false, fmt.Errorf("Pages deletion readback did not confirm absence (write may have applied)")
	}
	if err := c.checkLifecycleIdentity(repo, initial); err != nil {
		return false, fmt.Errorf("verify Pages deletion (write may have applied): %w", err)
	}
	return true, nil
}
func (c *PagesClient) createSite(repo repository.Repository, initial settings.ArchiveState, d settings.PagesDeclaration) (bool, error) {
	// ComparePages has already checked explicit source and creation prerequisites.
	payload := map[string]any{"build_type": *d.BuildType}
	if d.Source != nil {
		payload["source"] = *d.Source
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, &pagesPreWriteFailure{err}
	}
	if err := c.checkLifecycleIdentity(repo, initial); err != nil {
		return false, &pagesPreWriteFailure{err}
	}
	response, err := c.rest.Request(http.MethodPost, pagesPath(repo), bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("create Pages (write may have applied): %w", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		return false, fmt.Errorf("unexpected Pages create status %d (write may have applied)", response.StatusCode)
	}
	actual, err := c.Read(repo)
	if err != nil {
		return false, fmt.Errorf("verify Pages creation (write may have applied): %w", err)
	}
	sourceIntent := settings.PagesDeclaration{BuildType: d.BuildType, Source: d.Source}
	_, changes, err := settings.ComparePages(actual, sourceIntent)
	if err != nil || len(changes) != 0 {
		return false, fmt.Errorf("Pages creation readback did not confirm requested source (write may have applied)")
	}
	if err := c.checkLifecycleIdentity(repo, initial); err != nil {
		return false, fmt.Errorf("verify Pages creation (write may have applied): %w", err)
	}
	// Preserve the server's acquired defaults and visibility, and reconcile only
	// declared optional controls. Never retry or roll back a failed write.
	optional := settings.PagesDeclaration{CNAME: d.CNAME, HTTPSEnforced: d.HTTPSEnforced}
	if optional.Empty() {
		return true, nil
	}
	optionalWrote, err := c.reconcile(repo, optional, &initial)
	if err != nil {
		return true, &pagesCreatedFailure{err}
	}
	finalFailure := func(err error) (bool, error) {
		if !optionalWrote {
			err = &pagesPreWriteFailure{err}
		}
		return true, &pagesCreatedFailure{err}
	}
	final, err := c.Read(repo)
	if err != nil {
		return finalFailure(err)
	}
	_, changes, err = settings.ComparePages(final, d)
	finalState, stateErr := final.Values()
	createdState, _ := actual.Values()
	if err != nil || len(changes) != 0 || stateErr != nil || finalState.Public != createdState.Public {
		return finalFailure(fmt.Errorf("final Pages readback did not retain created source, declared controls or visibility"))
	}
	if err := c.checkLifecycleIdentity(repo, initial); err != nil {
		return finalFailure(err)
	}
	return true, nil
}
