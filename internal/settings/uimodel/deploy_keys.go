package uimodel

// Pointers distinguish required field presence from an explicit empty title or false grant.
type DeployKey struct {
	ID               *int64
	Title            *string
	Key              *string
	AllowWriteAccess *bool
}
