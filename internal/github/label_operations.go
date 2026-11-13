package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type labelPreWriteFailure struct{ cause error }

func (failure *labelPreWriteFailure) Error() string  { return failure.cause.Error() }
func (failure *labelPreWriteFailure) Unwrap() error  { return failure.cause }
func (*labelPreWriteFailure) ReadFailedBeforeWrite() {}

// CreateLabel confirms the complete supported object with an independent read.
// A positive acknowledgment ID remains available for recovery on failure.
func (client *Client) CreateLabel(repo repository.Repository, change settings.LabelChange) (int64, error) {
	if err := requireLabelOperation(change, "create"); err != nil {
		return 0, err
	}
	var err error
	var id int64
	change.Visit(func(after settings.LabelDeclaration) {
		payload := struct {
			Name        string  `json:"name"`
			Color       *string `json:"color,omitempty"`
			Description *string `json:"description,omitempty"`
		}{after.Name, after.Color, after.Description}
		body, encodeErr := json.Marshal(payload)
		if encodeErr != nil {
			err = encodeErr
			return
		}
		var response map[string]json.RawMessage
		err = client.rest.Post("repos/"+repo.Owner+"/"+repo.Name+"/labels", bytes.NewReader(body), &response)
		if err == nil {
			_ = json.Unmarshal(response["id"], &id)
			err = verifyLabelResponse(response, id, after)
			if err == nil {
				err = client.verifyLabelReadback(repo, id, after)
			}
		}
	}, nil, nil)
	return id, err
}

// UpdateLabel verifies the observed identity, sends the complete supported object,
// and independently reads its resulting name to confirm the same ID and settings.
func (client *Client) UpdateLabel(repo repository.Repository, change settings.LabelChange) error {
	if err := requireLabelOperation(change, "update"); err != nil {
		return err
	}
	var err error
	change.Visit(nil, func(before settings.LabelState, after settings.LabelDeclaration) {
		var fresh map[string]json.RawMessage
		if err = client.rest.Get(labelPath(repo, before.Name), &fresh); err != nil {
			err = &labelPreWriteFailure{cause: err}
			return
		}
		var freshID int64
		if json.Unmarshal(fresh["id"], &freshID) != nil || freshID != before.ID {
			err = &labelPreWriteFailure{cause: fmt.Errorf("label identity changed before update; refusing write")}
			return
		}
		payload := struct {
			Name        string  `json:"new_name"`
			Color       *string `json:"color,omitempty"`
			Description *string `json:"description,omitempty"`
		}{after.Name, after.Color, after.Description}
		body, encodeErr := json.Marshal(payload)
		if encodeErr != nil {
			err = encodeErr
			return
		}
		var response map[string]json.RawMessage
		err = client.rest.Patch(labelPath(repo, before.Name), bytes.NewReader(body), &response)
		if err == nil {
			err = verifyLabelResponse(response, before.ID, after)
			if err == nil {
				err = client.verifyLabelReadback(repo, before.ID, after)
			}
		}
	}, nil)
	return err
}

func (client *Client) verifyLabelReadback(repo repository.Repository, id int64, expected settings.LabelDeclaration) error {
	var response map[string]json.RawMessage
	if err := client.rest.Get(labelPath(repo, expected.Name), &response); err != nil {
		return fmt.Errorf("read back label (write may have applied): %w", err)
	}
	if err := verifyLabelResponse(response, id, expected); err != nil {
		return fmt.Errorf("read back label (write may have applied): %w", err)
	}
	return nil
}

func verifyLabelResponse(response map[string]json.RawMessage, id int64, expected settings.LabelDeclaration) error {
	var actualID int64
	var name, color string
	var description *string
	if id <= 0 || json.Unmarshal(response["id"], &actualID) != nil || actualID != id ||
		json.Unmarshal(response["name"], &name) != nil || settings.LabelIdentity(name) != settings.LabelIdentity(expected.Name) ||
		json.Unmarshal(response["color"], &color) != nil || expected.Color == nil || !strings.EqualFold(color, *expected.Color) ||
		json.Unmarshal(response["description"], &description) != nil || expected.Description == nil {
		return fmt.Errorf("GitHub returned incomplete or mismatching label identity/settings (writes may already have applied)")
	}
	actualDescription := ""
	if description != nil {
		actualDescription = *description
	}
	if actualDescription != *expected.Description {
		return fmt.Errorf("GitHub returned mismatching label description (writes may already have applied)")
	}
	return nil
}

// DeleteLabel checks the exact acquired object before deletion and confirms
// absence using the complete, paginated inventory. No retry restores assignments.
func (client *Client) DeleteLabel(repo repository.Repository, change settings.LabelChange) error {
	if err := requireLabelOperation(change, "delete"); err != nil {
		return err
	}
	var result error
	change.Visit(nil, nil, func(before settings.LabelState) {
		var fresh map[string]json.RawMessage
		if err := client.rest.Get(labelPath(repo, before.Name), &fresh); err != nil {
			result = &labelPreWriteFailure{cause: err}
			return
		}
		var actualName string
		if json.Unmarshal(fresh["name"], &actualName) != nil || actualName != before.Name {
			result = &labelPreWriteFailure{cause: fmt.Errorf("label name changed before deletion; refusing write")}
			return
		}
		expected := settings.LabelDeclaration{ID: &before.ID, Name: before.Name, Color: &before.Color, Description: &before.Description}
		if err := verifyLabelResponse(fresh, before.ID, expected); err != nil {
			result = &labelPreWriteFailure{cause: fmt.Errorf("label identity or state changed before deletion; refusing write: %w", err)}
			return
		}
		response, err := client.rest.Request(http.MethodDelete, labelPath(repo, before.Name), nil)
		if err != nil {
			result = fmt.Errorf("delete label (write may have applied): %w", err)
			return
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			result = fmt.Errorf("unexpected label delete status %d (write may have applied)", response.StatusCode)
			return
		}
		observed, err := client.ReadLabels(repo)
		if err != nil {
			result = fmt.Errorf("verify label deletion (write may have applied): %w", err)
			return
		}
		values, err := observed.Values()
		if err != nil {
			result = err
			return
		}
		for _, value := range values {
			if value.ID == before.ID || settings.LabelIdentity(value.Name) == settings.LabelIdentity(before.Name) {
				result = fmt.Errorf("label deletion readback retained the identity or name (write may have applied)")
				return
			}
		}
	})
	return result
}

func requireLabelOperation(change settings.LabelChange, operation string) error {
	if change == (settings.LabelChange{}) || change.Operation() != operation {
		return fmt.Errorf("expected %q label operation", operation)
	}
	return nil
}
