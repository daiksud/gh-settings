// Package uimodel prepares UI-oriented values for the future public contract.
// It is internal preparation; the current YAML codec does not consume it.
package uimodel

// About uses the repository's About dialog labels. Nil means unmanaged.
type About struct {
	Description *string
	Website     *string
	Topics      *[]string
}
