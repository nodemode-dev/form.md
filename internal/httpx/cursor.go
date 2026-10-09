package httpx

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
)

type Cursor struct {
	Offset int    `json:"o,omitempty"`
	At     int64  `json:"t,omitempty"`
	ID     string `json:"i,omitempty"`
}

func EncodeCursor(c Cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func DecodeCursor(s string) (Cursor, error) {
	var c Cursor
	if s == "" {
		return c, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || json.Unmarshal(raw, &c) != nil || c.Offset < 0 {
		return Cursor{}, Fail(http.StatusBadRequest, "bad_cursor", "the cursor is not one this server gave out")
	}
	return c, nil
}
