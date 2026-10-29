package uimodel

// Repository groups UI-oriented configuration beside its About dialog.
type Repository struct {
	About    *About
	Labels   *[]Label
	Settings *RepositorySettings
}

type RepositorySettings struct {
	General *General
}
