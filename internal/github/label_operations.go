package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type labelPreWriteFailure struct{ cause error }

func (failure *labelPreWriteFailure) Error() string  { return failure.cause.Error() }
func (failure *labelPreWriteFailure) Unwrap() error  { return failure.cause }
func (*labelPreWriteFailure) ReadFailedBeforeWrite() {}

// CreateLabel sends only the attributes retained by a typed create operation.
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
		}
	}, nil, nil)
	return id, err
}

// UpdateLabel verifies the observed identity and sends the complete supported object.
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
		}
	}, nil)
	return err
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

// DeleteLabel deletes the exact name retained by a typed delete operation.
func (client *Client) DeleteLabel(repo repository.Repository, change settings.LabelChange) error {
	if err := requireLabelOperation(change, "delete"); err != nil {
		return err
	}
	return client.rest.Delete(labelPath(repo, change.Name()), nil)
}

func requireLabelOperation(change settings.LabelChange, operation string) error {
	if change == (settings.LabelChange{}) || change.Operation() != operation {
		return fmt.Errorf("expected %q label operation", operation)
	}
	return nil
}
