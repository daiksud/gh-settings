package settings

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// UnmarshalJSON never turns absent/null raw members into an acquired empty pair.
// Closed menu validation remains the decision constructor's responsibility.
func (normalization *CommitMessageNormalization) UnmarshalJSON(data []byte) error {
	var fields struct {
		Path       *string                      `json:"path"`
		RawTitle   *string                      `json:"raw_title"`
		RawMessage *string                      `json:"raw_message"`
		Fallback   *uimodel.CommitMessageChoice `json:"fallback"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("decode commit message normalization: %w", err)
	}
	if fields.Path == nil || fields.RawTitle == nil || fields.RawMessage == nil || fields.Fallback == nil {
		return fmt.Errorf("commit message normalization requires string path, raw_title, raw_message and fallback")
	}
	*normalization = CommitMessageNormalization{Path: *fields.Path, RawTitle: *fields.RawTitle, RawMessage: *fields.RawMessage, Fallback: *fields.Fallback}
	return nil
}
