package settings

// ArchivedCopilotToggles holds independently acquired settings. Nil means no
// value was acquired; in particular, it is not an observed false.
type ArchivedCopilotToggles struct {
	Instructions *bool
	MCP          *bool
}
