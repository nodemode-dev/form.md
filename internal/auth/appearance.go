package auth

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/nodemode-ai/form.md/internal/httpx"
)

type Appearance struct {
	Mode     string `json:"mode"`
	Dark     string `json:"dark"`
	Light    string `json:"light"`
	Accent   string `json:"accent"`
	Density  string `json:"density"`
	Contrast string `json:"contrast"`
	At       int64  `json:"at"`
}

var (
	lookName     = regexp.MustCompile(`^[a-z]{0,12}$`)
	lookAccent   = regexp.MustCompile(`^(#[0-9a-f]{6})?$`)
	lookMode     = map[string]bool{"": true, "auto": true, "light": true, "dark": true}
	lookDensity  = map[string]bool{"": true, "compact": true, "standard": true, "comfortable": true}
	lookContrast = map[string]bool{"": true, "auto": true, "standard": true, "high": true}
)

func (a *Appearance) encode() (string, error) {
	if !lookMode[a.Mode] || !lookName.MatchString(a.Dark) || !lookName.MatchString(a.Light) || !lookAccent.MatchString(a.Accent) ||
		!lookDensity[a.Density] || !lookContrast[a.Contrast] || a.At < 0 {
		return "", httpx.Fail(http.StatusBadRequest, "bad_appearance", "unsupported appearance value")
	}
	b, err := json.Marshal(a)
	return string(b), err
}

func decodeAppearance(raw string) *Appearance {
	if raw == "" {
		return nil
	}
	var a Appearance
	if json.Unmarshal([]byte(raw), &a) != nil {
		return nil
	}
	return &a
}
