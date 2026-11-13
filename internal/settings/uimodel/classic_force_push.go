package uimodel

type ClassicForcePush struct {
	Enabled         *bool                    `json:"enabled,omitempty"`
	WhoCanForcePush *string                  `json:"who_can_force_push,omitempty"`
	Actors          *[]ClassicForcePushActor `json:"actors,omitempty"`
}

type ClassicForcePushActor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
