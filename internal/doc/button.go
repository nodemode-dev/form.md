package doc

import (
	"encoding/base64"
	"encoding/json"
)

type buttonSpec struct {
	Label   string          `json:"label"`
	Actions json.RawMessage `json:"actions,omitempty"`
}

func buttonEncode(label, actions string) string {
	spec := buttonSpec{Label: label}
	if json.Valid([]byte(actions)) {
		spec.Actions = json.RawMessage(actions)
	}
	b, _ := json.Marshal(spec)
	return base64.RawURLEncoding.EncodeToString(b)
}

func buttonNode(enc string) *pmNode {
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil || len(raw) > 64<<10 {
		return nil
	}
	var spec buttonSpec
	if json.Unmarshal(raw, &spec) != nil {
		return nil
	}
	n := newNode("button")
	n.attrs["label"] = spec.Label
	n.attrs["actions"] = "[]"
	if len(spec.Actions) > 0 {
		n.attrs["actions"] = string(spec.Actions)
	}
	return n
}
