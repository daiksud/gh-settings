package uimodel

type OIDCSettings struct{ SubjectClaim *OIDCSubjectTemplate }
type OIDCSubjectTemplate struct {
	UseDefaultTemplate   *bool     `json:"use_default_template"`
	SubjectClaimTemplate *[]string `json:"subject_claim_template,omitempty"`
}
