package pagestyle

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/nodemode-ai/form.md/internal/httpx"
)

type Style struct {
	Font    string `json:"font,omitempty"`
	Size    string `json:"size,omitempty"`
	Width   string `json:"width,omitempty"`
	Spacing string `json:"spacing,omitempty"`
	Align   string `json:"align,omitempty"`
	Header  string `json:"header,omitempty"`
}

var allowed = map[string]map[string]bool{
	"font":    {"": true, "sans": true, "serif": true, "mono": true, "rounded": true, "slab": true, "humanist": true},
	"size":    {"": true, "small": true, "default": true, "large": true},
	"width":   {"": true, "default": true, "wide": true, "full": true},
	"spacing": {"": true, "compact": true, "default": true, "relaxed": true},
	"align":   {"": true, "start": true, "justify": true, "center": true},
	"header":  {"": true, "left": true, "center": true, "right": true},
}

var errBad = httpx.Fail(http.StatusBadRequest, "bad_style", "a page style takes font (sans, serif, mono, rounded, slab, humanist), size (small, default, large), width (default, wide, full), spacing (compact, default, relaxed), align (start, justify, center) and header (left, center, right)")

func Clean(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var s Style
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", errBad
	}
	for k, v := range map[string]string{"font": s.Font, "size": s.Size, "width": s.Width, "spacing": s.Spacing, "align": s.Align, "header": s.Header} {
		if k == "font" && fontRef.MatchString(v) {
			continue
		}
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

var fontRef = regexp.MustCompile(`^f:[a-z0-9]{26}$`)
var familyOK = regexp.MustCompile(`^[\p{L}\p{N} ._-]{1,60}$`)

const MaxFonts = 24

type Font struct {
	File   string `json:"file"`
	Family string `json:"family"`
}

func CleanFonts(in []Font, owned func(ref string) bool) (string, error) {
	if len(in) > MaxFonts {
		return "", httpx.Fail(http.StatusBadRequest, "too_many_fonts", "a workspace can keep up to 24 fonts")
	}
	seen := map[string]bool{}
	out := make([]Font, 0, len(in))
	for _, f := range in {
		fam := strings.TrimSpace(f.Family)
		if !fontRef.MatchString(f.File) || !familyOK.MatchString(fam) {
			return "", httpx.Fail(http.StatusBadRequest, "bad_font", "a font is an uploaded font file and a name of letters, digits, spaces, dots, dashes or underscores")
		}
		if seen[f.File] {
			continue
		}
		if !owned(f.File) {
			return "", httpx.Fail(http.StatusBadRequest, "bad_font", "upload the font to this workspace first")
		}
		seen[f.File] = true
		out = append(out, Font{File: f.File, Family: fam})
	}
	if len(out) == 0 {
		return "", nil
	}
	raw, _ := json.Marshal(out)
	return string(raw), nil
}

type Preset struct {
	Name  string `json:"name"`
	Style Style  `json:"style"`
}

const MaxPresets = 24

var presetName = regexp.MustCompile(`^[\p{L}\p{N} ._'&()-]{1,60}$`)

func CleanPresets(in []Preset) (string, error) {
	if len(in) > MaxPresets {
		return "", httpx.Fail(http.StatusBadRequest, "too_many_presets", "a workspace can keep up to 24 saved styles")
	}
	seen := map[string]bool{}
	out := make([]Preset, 0, len(in))
	for _, p := range in {
		name := strings.TrimSpace(p.Name)
		if !presetName.MatchString(name) {
			return "", httpx.Fail(http.StatusBadRequest, "bad_preset", "a saved style needs a name of up to 60 letters, digits, spaces or simple punctuation")
		}
		raw, _ := json.Marshal(p.Style)
		clean, err := Clean(raw)
		if err != nil {
			return "", err
		}
		var st Style
		_ = json.Unmarshal([]byte(clean), &st)
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Preset{Name: name, Style: st})
	}
	if len(out) == 0 {
		return "", nil
	}
	raw, _ := json.Marshal(out)
	return string(raw), nil
}

func ParsePresets(stored string) []Preset {
	var out []Preset
	if stored != "" {
		_ = json.Unmarshal([]byte(stored), &out)
	}
	return out
}

func ParseFonts(stored string) []Font {
	var out []Font
	if stored != "" {
		_ = json.Unmarshal([]byte(stored), &out)
	}
	return out
}

func FontsCSS(fonts []Font) string {
	var b strings.Builder
	for _, f := range fonts {
		id := strings.TrimPrefix(f.File, "f:")
		b.WriteString("@font-face{font-family:\"wsf-" + id + "\";src:url(\"/files/" + id + "\");font-display:block}\n")
		b.WriteString(":root[data-style-font=\"" + f.File + "\"] .view--page .page{font-family:\"wsf-" + id + "\",var(--font-sans)}\n")
	}
	return b.String()
}

func Merge(base, over string) Style {
	var b, o Style
	_ = json.Unmarshal([]byte(base), &b)
	_ = json.Unmarshal([]byte(over), &o)
	pick := func(x, y string) string {
		if y != "" {
			return y
		}
		return x
	}
	return Style{Font: pick(b.Font, o.Font), Size: pick(b.Size, o.Size), Width: pick(b.Width, o.Width), Spacing: pick(b.Spacing, o.Spacing), Align: pick(b.Align, o.Align), Header: pick(b.Header, o.Header)}
}

func (s Style) Attrs() [][2]string {
	var out [][2]string
	for _, kv := range [][2]string{{"font", s.Font}, {"size", s.Size}, {"width", s.Width}, {"spacing", s.Spacing}, {"align", s.Align}, {"header", s.Header}} {
		if kv[1] != "" && kv[1] != "default" && kv[1] != "start" && kv[1] != "sans" && kv[1] != "left" && (allowed[kv[0]][kv[1]] || (kv[0] == "font" && fontRef.MatchString(kv[1]))) {
			out = append(out, kv)
		}
	}
	return out
}

func Raw(stored string) json.RawMessage {
	if stored == "" {
		return nil
	}
	return json.RawMessage(stored)
}
