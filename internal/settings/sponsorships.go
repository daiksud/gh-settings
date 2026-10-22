package settings

// SponsorshipsState contains one complete, independently acquired display setting.
// Funding destinations are outside this observation and declaration.
type SponsorshipsState struct{ Enabled bool }

var sponsorshipBoolFields = [...]repositoryField[SponsorshipsState, bool]{
	{
		name:       "repository.has_sponsorships",
		current:    func(state SponsorshipsState) bool { return state.Enabled },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.HasSponsorships },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.HasSponsorships = &value },
	},
}

// WithSponsorships adds the acquired boolean without inventing other observations.
func (observation RepositoryObservation) WithSponsorships(state SponsorshipsState) RepositoryObservation {
	declareRepositoryFields(sponsorshipBoolFields[:], state, &observation.values)
	return observation
}
