package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// CreateLabel sends only the attributes retained by a typed create operation.
func (client *Client) CreateLabel(repo repository.Repository, change settings.LabelChange) error {
	if err := requireLabelOperation(change, "create"); err != nil {
		return err
	}
	var err error
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
		err = client.rest.Post("repos/"+repo.Owner+"/"+repo.Name+"/labels", bytes.NewReader(body), nil)
	}, nil, nil)
	return err
}

// UpdateLabel sends only changed declared attributes from a typed update.
func (client *Client) UpdateLabel(repo repository.Repository, change settings.LabelChange) error {
	if err := requireLabelOperation(change, "update"); err != nil {
		return err
	}
	var err error
	change.Visit(nil, func(before settings.LabelState, after settings.LabelDeclaration) {
		payload := struct {
			Color       *string `json:"color,omitempty"`
			Description *string `json:"description,omitempty"`
		}{after.Color, after.Description}
		body, encodeErr := json.Marshal(payload)
		if encodeErr != nil {
			err = encodeErr
			return
		}
		err = client.rest.Patch(labelPath(repo, before.Name), bytes.NewReader(body), nil)
	}, nil)
	return err
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
