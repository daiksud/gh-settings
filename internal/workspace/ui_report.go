package workspace

import (
	"encoding/json"
	"reflect"
)

// MarshalUIReport uses the configuration codec's field names and omission
// semantics to encode an already projected UI model for display.
func MarshalUIReport(value any) ([]byte, error) {
	node, err := encodeUIControl(reflect.ValueOf(value))
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := node.Decode(&decoded); err != nil {
		return nil, err
	}
	return json.Marshal(decoded)
}
