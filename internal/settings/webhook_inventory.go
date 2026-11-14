package settings

import "fmt"

func (o WebhookObservation) ValidateComplete() error {
	if !o.Complete {
		return fmt.Errorf("complete webhook inventory required")
	}
	return o.Validate()
}
func (o WebhookObservation) Clone() WebhookObservation {
	if o.Values != nil {
		values := make([]WebhookState, len(o.Values))
		for i, v := range o.Values {
			values[i] = v.Clone()
		}
		o.Values = values
	}
	return o
}
func (o WebhookObservation) Find(id int64) (WebhookState, bool) {
	for _, s := range o.Values {
		if s.ID == id {
			return s.Clone(), true
		}
	}
	return WebhookState{}, false
}
func WebhookStateEqual(a, b WebhookState) bool {
	return WebhookIdentityMatches(a, b) && a.Config == b.Config && a.Active == b.Active && WebhookEventsEqual(a.Events, b.Events)
}
func VerifyWebhookInventory(expected, actual WebhookObservation) error {
	if expected.ValidateComplete() != nil || actual.ValidateComplete() != nil || expected.Target != actual.Target || len(expected.Values) != len(actual.Values) {
		return fmt.Errorf("webhook inventory or repository identity changed")
	}
	for _, v := range expected.Values {
		a, ok := actual.Find(v.ID)
		if !ok || !WebhookStateEqual(v, a) {
			return fmt.Errorf("webhook inventory state changed")
		}
	}
	return nil
}
func (o WebhookObservation) WithUpdated(d Webhook) (WebhookObservation, error) {
	if o.ValidateComplete() != nil || d.Validate() != nil || d.ID <= 0 {
		return WebhookObservation{}, fmt.Errorf("invalid webhook inventory update")
	}
	o = o.Clone()
	for i, s := range o.Values {
		if s.ID == d.ID {
			if !s.GuardsMatch(d) {
				return WebhookObservation{}, fmt.Errorf("webhook parent guard differs")
			}
			o.Values[i].Config = s.Config.Overlay(d)
			return o, nil
		}
	}
	return WebhookObservation{}, fmt.Errorf("updated webhook identity missing")
}
func (o WebhookObservation) WithCreated(s WebhookState) (WebhookObservation, error) {
	if o.ValidateComplete() != nil || s.Validate() != nil {
		return WebhookObservation{}, fmt.Errorf("invalid created webhook state")
	}
	if _, ok := o.Find(s.ID); ok {
		return WebhookObservation{}, fmt.Errorf("created webhook identity already exists")
	}
	o = o.Clone()
	o.Values = append(o.Values, s.Clone())
	return o, nil
}
func WebhookCreationMatches(s WebhookState, d Webhook) bool {
	return d.ID == 0 && d.Validate() == nil && s.Validate() == nil && !s.Config.InsecureSSL && s.Config.PayloadURL == *d.PayloadURL && s.Config.ContentType == *d.ContentType && s.Active == *d.Active && WebhookEventsEqual(s.Events, *d.Events)
}

type WebhookCreateResult struct {
	Attempted   bool
	RecoveryID  int64
	Observation WebhookObservation // Complete only after independent verification.
}
