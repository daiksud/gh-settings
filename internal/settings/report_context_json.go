package settings

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func (p *CommitMessagePair) UnmarshalJSON(data []byte) error {
	var raw struct {
		Title   *string `json:"title"`
		Message *string `json:"message"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return fmt.Errorf("decode commit message pair: %w", err)
	}
	if raw.Title == nil || raw.Message == nil {
		return fmt.Errorf("commit message pair requires title and message")
	}
	*p = CommitMessagePair{Title: *raw.Title, Message: *raw.Message}
	return nil
}

func (c *CommitMessageContext) UnmarshalJSON(data []byte) error {
	var raw struct {
		Menu   *CommitMessageMenu `json:"menu"`
		Before *CommitMessagePair `json:"before"`
		After  *CommitMessagePair `json:"after"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return fmt.Errorf("decode commit message context: %w", err)
	}
	if raw.Menu == nil || raw.Before == nil || raw.After == nil {
		return fmt.Errorf("commit message context requires menu and pairs")
	}
	value := CommitMessageContext{Menu: *raw.Menu, Before: *raw.Before, After: *raw.After}
	if _, _, err := value.Choices(); err != nil {
		return err
	}
	*c = value
	return nil
}

func (c *ActionsExecutionContext) UnmarshalJSON(data []byte) error {
	var raw struct {
		Before          *uimodel.ActionsPermissionChoice `json:"before"`
		After           *uimodel.ActionsPermissionChoice `json:"after"`
		ObservedAllowed *string                          `json:"observed_allowed"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return fmt.Errorf("decode Actions execution context: %w", err)
	}
	if raw.Before == nil || raw.After == nil || raw.ObservedAllowed == nil {
		return fmt.Errorf("Actions execution context requires both choices")
	}
	value := ActionsExecutionContext{Before: *raw.Before, After: *raw.After, ObservedAllowed: *raw.ObservedAllowed}
	if err := value.Validate(); err != nil {
		return err
	}
	*c = value
	return nil
}
