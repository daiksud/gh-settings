package uimodel

// Label uses the Labels editor's fields. ID retains selected-object ownership;
// a missing ID requests creation. Color includes the displayed # prefix.
type Label struct {
	ID                 *int64
	Name               string
	Color, Description *string
}
