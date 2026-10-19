package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

// LabelAttributes holds only the writable attributes represented by this report.
type LabelAttributes struct {
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (attributes *LabelAttributes) UnmarshalJSON(data []byte) error {
	type plain LabelAttributes
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for key, value := range raw {
		if key != "color" && key != "description" {
			return fmt.Errorf("unknown label attribute %q", key)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("null label attribute %q", key)
		}
	}
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*attributes = LabelAttributes(value)
	return nil
}

// LabelPlanChange is same-version presentation data, never apply input.
type LabelPlanChange struct {
	Operation string           `json:"operation"`
	Name      string           `json:"name"`
	Before    *LabelAttributes `json:"before,omitempty"`
	After     *LabelAttributes `json:"after,omitempty"`
}

func (change *LabelPlanChange) UnmarshalJSON(data []byte) error {
	type plain LabelPlanChange
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*change = LabelPlanChange(value)
	return nil
}

func labelPlanChange(change settings.LabelChange) LabelPlanChange {
	result := LabelPlanChange{Operation: change.Operation(), Name: change.Name()}
	before := func(value settings.LabelState) {
		color, description := value.Color, value.Description
		result.Before = &LabelAttributes{Color: &color, Description: &description}
	}
	after := func(value settings.LabelDeclaration) {
		result.After = &LabelAttributes{Color: value.Color, Description: value.Description}
	}
	change.Visit(after, func(prior settings.LabelState, declared settings.LabelDeclaration) { before(prior); after(declared) }, before)
	return result
}

func validateLabelReport(changes []LabelPlanChange) (settings.LabelChanges, error) {
	native := make([]settings.LabelChange, 0, len(changes))
	for _, change := range changes {
		var before settings.LabelState
		var after settings.LabelDeclaration
		if change.Before != nil {
			if change.Before.Color == nil || change.Before.Description == nil {
				return settings.LabelChanges{}, fmt.Errorf("incomplete label before value")
			}
			before = settings.LabelState{Name: change.Name, Color: *change.Before.Color, Description: *change.Before.Description}
		}
		if change.After != nil {
			after = settings.LabelDeclaration{Name: change.Name, Color: change.After.Color, Description: change.After.Description}
		}
		var value settings.LabelChange
		var err error
		switch change.Operation {
		case "create":
			if change.Before != nil || change.After == nil {
				return settings.LabelChanges{}, fmt.Errorf("invalid %q label operation", change.Operation)
			}
			value, err = settings.NewLabelCreate(after)
		case "update":
			if change.Before == nil || change.After == nil {
				return settings.LabelChanges{}, fmt.Errorf("invalid %q label operation", change.Operation)
			}
			value, err = settings.NewLabelUpdate(before, after)
		case "delete":
			if change.Before == nil || change.After != nil {
				return settings.LabelChanges{}, fmt.Errorf("invalid %q label operation", change.Operation)
			}
			value, err = settings.NewLabelDelete(before)
		default:
			return settings.LabelChanges{}, fmt.Errorf("invalid %q label operation", change.Operation)
		}
		if err != nil {
			return settings.LabelChanges{}, err
		}
		native = append(native, value)
	}
	return settings.NewLabelChanges(native)
}

func labelSummaryLine(change settings.LabelChange) string {
	prefix := fmt.Sprintf("settings.labels: %s %q", change.Operation(), change.Name())
	change.Visit(func(after settings.LabelDeclaration) {
		var attributes []string
		if after.Color != nil {
			attributes = append(attributes, fmt.Sprintf("color=%q", *after.Color))
		}
		if after.Description != nil {
			attributes = append(attributes, fmt.Sprintf("description=%q", *after.Description))
		}
		if len(attributes) > 0 {
			prefix += " " + strings.Join(attributes, " ")
		}
	}, func(before settings.LabelState, after settings.LabelDeclaration) {
		var attributes []string
		if after.Color != nil {
			attributes = append(attributes, fmt.Sprintf("color: %q → %q", before.Color, *after.Color))
		}
		if after.Description != nil {
			attributes = append(attributes, fmt.Sprintf("description: %q → %q", before.Description, *after.Description))
		}
		prefix += " " + strings.Join(attributes, ", ")
	}, func(settings.LabelState) { prefix += " (removes issue/PR associations)" })
	return prefix
}
