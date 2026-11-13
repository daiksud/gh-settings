package uimodel

type OIDCSettings struct{ SubjectClaim *OIDCSubjectTemplate }
type OIDCSubjectTemplate struct {
	UseDefaultTemplate       *bool     `json:"use_default_template,omitempty"`
	SubjectClaimTemplate     *[]string `json:"subject_claim_template,omitempty"`
	UseImmutableSubjectClaim *bool     `json:"use_immutable_subject_claim,omitempty"`
}
