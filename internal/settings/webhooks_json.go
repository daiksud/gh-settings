package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func webhookReportObject(data []byte, allowed map[string]bool) (map[string]json.RawMessage, error) {
	invalid := func() (map[string]json.RawMessage, error) { return nil, fmt.Errorf("invalid webhook report member") }
	d := json.NewDecoder(bytes.NewReader(data))
	t, e := d.Token()
	if e != nil || t != json.Delim('{') {
		return invalid()
	}
	result := map[string]json.RawMessage{}
	for d.More() {
		t, e = d.Token()
		if e != nil {
			return invalid()
		}
		k, ok := t.(string)
		if !ok || !allowed[k] || result[k] != nil {
			return invalid()
		}
		var v json.RawMessage
		if d.Decode(&v) != nil || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return invalid()
		}
		result[k] = v
	}
	if t, e = d.Token(); e != nil || t != json.Delim('}') {
		return invalid()
	}
	if _, e = d.Token(); e != io.EOF {
		return invalid()
	}
	return result, nil
}
func (v *Webhook) UnmarshalJSON(data []byte) error {
	raw, err := webhookReportObject(data, map[string]bool{"id": true, "payload_url": true, "content_type": true, "active": true, "events": true})
	if err != nil {
		return err
	}
	if id, present := raw["id"]; present {
		var number int64
		if json.Unmarshal(id, &number) != nil || number <= 0 {
			return fmt.Errorf("invalid webhook report identity")
		}
	}
	type plain Webhook
	var value plain
	if json.Unmarshal(data, &value) != nil {
		return fmt.Errorf("invalid webhook report value")
	}
	if err = Webhook(value).Validate(); err != nil {
		return err
	}
	*v = Webhook(value)
	return nil
}
func (c *WebhookChange) UnmarshalJSON(data []byte) error {
	raw, err := webhookReportObject(data, map[string]bool{"before": true, "after": true})
	if err != nil {
		return err
	}
	if raw["after"] == nil {
		return fmt.Errorf("webhook report requires desired projection")
	}
	type plain WebhookChange
	var value plain
	if json.Unmarshal(data, &value) != nil {
		return fmt.Errorf("invalid webhook report projection")
	}
	if err = ValidateWebhookChanges([]WebhookChange{WebhookChange(value)}); err != nil {
		return err
	}
	*c = WebhookChange(value)
	return nil
}
