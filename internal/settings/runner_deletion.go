package settings

import (
	"fmt"
	"strings"
)

// RunnerDeletion is the public, non-operational projection of a fixed candidate.
type RunnerDeletion struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	OS           string   `json:"os"`
	CustomLabels []string `json:"custom_labels"`
}

func ValidateRunnerDeletions(values []RunnerDeletion) error {
	seen := map[int64]bool{}
	for _, v := range values {
		if v.ID <= 0 || seen[v.ID] || strings.TrimSpace(v.Name) == "" || strings.TrimSpace(v.OS) == "" || v.CustomLabels == nil {
			return fmt.Errorf("invalid runner deletion")
		}
		seen[v.ID] = true
		labels := v.CustomLabels
		if err := ValidateRunnerLabels([]RunnerLabels{{ID: v.ID, CustomLabels: &labels}}); err != nil {
			return err
		}
	}
	return nil
}
