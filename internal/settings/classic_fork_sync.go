package settings

// Inactive native values remain private companions, not writable defaults.
func ClassicForkSyncProjection(isFork, locked, allows bool) *bool {
	if !isFork || !locked {
		return nil
	}
	return &allows
}
