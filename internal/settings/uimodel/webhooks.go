package uimodel

type Webhook struct {
	ID          *int64
	PayloadURL  *string
	ContentType *string
	Active      *bool
	Events      *[]string
}
