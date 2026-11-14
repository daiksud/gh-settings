package uimodel

type Webhook struct {
	ID              *int64
	PayloadURL      *string
	ContentType     *string
	SSLVerification *bool
	Active          *bool
	Events          *[]string
}
