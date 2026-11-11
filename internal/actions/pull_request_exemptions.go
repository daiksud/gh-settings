package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

// PullRequestExemptionsPlanChange is presentation data for one complete membership change.
type PullRequestExemptionsPlanChange struct {
	Before []string `json:"before"`
	After  []string `json:"after"`
}

func (c *PullRequestExemptionsPlanChange) UnmarshalJSON(data []byte) error {
	type plain PullRequestExemptionsPlanChange
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if value.Before == nil || value.After == nil {
		return fmt.Errorf("missing exemption list report membership")
	}
	*c = PullRequestExemptionsPlanChange(value)
	return nil
}
func validatePullRequestExemptionsReport(raw []PullRequestExemptionsPlanChange) ([]settings.PullRequestExemptionsChange, error) {
	if len(raw) > 1 {
		return nil, fmt.Errorf("duplicate exemption list report")
	}
	var result []settings.PullRequestExemptionsChange
	for _, entry := range raw {
		if entry.Before == nil || entry.After == nil {
			return nil, fmt.Errorf("missing exemption list report membership")
		}
		change, err := settings.NewPullRequestExemptionsChange(entry.Before, entry.After)
		if err != nil {
			return nil, err
		}
		result = append(result, change)
	}
	return result, nil
}
