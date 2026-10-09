package server

import (
	"html"
	"strings"

	"github.com/nodemode-ai/form.md/internal/oauth"
)

func consentChoices(v *oauth.View) (string, string) {
	var scope strings.Builder
	option := func(value, title, hint string, checked, disabled bool) {
		scope.WriteString(`<label class="option-card"><input type="radio" name="scope" value="` + value + `"`)
		if checked {
			scope.WriteString(" checked")
		}
		if disabled {
			scope.WriteString(" disabled")
		}
		scope.WriteString(`><span class="option-card__title">` + html.EscapeString(title) + `</span><span class="option-card__hint">` + html.EscapeString(hint) + `</span></label>`)
	}
	write := v.Scope == "write"
	option("write", "Read and edit", "Pages, databases, rows and comments, as you could.", write, !write)
	option("read", "Read only", "Search and read; no changes.", !write, false)
	var ws strings.Builder
	for _, w := range v.Workspaces {
		name := w.Name
		if w.Personal {
			name += " (personal)"
		}
		ws.WriteString(`<label class="check"><input type="checkbox" name="workspace" value="` + html.EscapeString(w.ID) + `" checked><span>` + html.EscapeString(name) + `</span></label>`)
	}
	return scope.String(), ws.String()
}
