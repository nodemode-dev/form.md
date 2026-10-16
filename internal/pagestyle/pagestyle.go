package pagestyle

import (
	"encoding/json"
	"net/http"

	"github.com/nodemode-ai/form.md/internal/httpx"
)

type Style struct {
	Font    string `json:"font,omitempty"`
	Size    string `json:"size,omitempty"`
	Width   string `json:"width,omitempty"`
	Spacing string `json:"spacing,omitempty"`
	Align   string `json:"align,omitempty"`
}

var allowed = map[string]map[string]bool{
	"font":    {"": true, "sans": true, "serif": true, "mono": true, "rounded": true, "slab": true, "humanist": true},
	"size":    {"": true, "small": true, "default": true, "large": true},
	"width":   {"": true, "default": true, "wide": true, "full": true},
	"spacing": {"": true, "compact": true, "default": true, "relaxed": true},
	"align":   {"": true, "start": true, "justify": true, "center": true},
}

var errBad = httpx.Fail(http.StatusBadRequest, "bad_style", "a page style takes font (sans, serif, mono, rounded, slab, humanist), size (small, default, large), width (default, wide, full), spacing (compact, default, relaxed) and align (start, justify, center)")

func Clean(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var s Style
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", errBad
	}
	for k, v := range map[string]string{"font": s.Font, "size": s.Size, "width": s.Width, "spacing": s.Spacing, "align": s.Align} {
		if !allowed[k][v] {
			return "", errBad
		}
	}
	if s == (Style{}) {
		return "", nil
	}
	out, _ := json.Marshal(s)
	return string(out), nil
}

func Raw(stored string) json.RawMessage {
	if stored == "" {
		return nil
	}
	return json.RawMessage(stored)
}
