package static

import (
	"bytes"
	"html"
	"regexp"
	"strings"
	"sync"

	"github.com/nodemode-ai/form.md/internal/i18n"
)

var (
	textNode  = regexp.MustCompile(`>([^<>]+)<`)
	attrText  = regexp.MustCompile(`(aria-label|placeholder|title|data-tip)="([^"]+)"`)
	rawBlock  = regexp.MustCompile(`(?s)<(script|style)\b.*?</(script|style)>`)
	localized sync.Map
)

func translate(lang, s string) (string, bool) {
	core := strings.TrimSpace(html.UnescapeString(s))
	if core == "" {
		return s, false
	}
	v, ok := i18n.Has(lang, core)
	if !ok || v == "" {
		return s, false
	}
	lead := s[:len(s)-len(strings.TrimLeft(s, " \t\r\n"))]
	trail := s[len(strings.TrimRight(s, " \t\r\n")):]
	return lead + html.EscapeString(v) + trail, true
}

func localize(index []byte, lang string) []byte {
	if i18n.Base(lang) == "en" {
		return index
	}
	if v, ok := localized.Load(lang); ok {
		return v.([]byte)
	}
	var out bytes.Buffer
	last := 0
	for _, loc := range rawBlock.FindAllIndex(index, -1) {
		out.Write(localizeChunk(index[last:loc[0]], lang))
		out.Write(index[loc[0]:loc[1]])
		last = loc[1]
	}
	out.Write(localizeChunk(index[last:], lang))
	b := out.Bytes()
	localized.Store(lang, b)
	return b
}

func localizeChunk(b []byte, lang string) []byte {
	b = textNode.ReplaceAllFunc(b, func(m []byte) []byte {
		inner := string(m[1 : len(m)-1])
		if strings.Contains(inner, "<!--") {
			return m
		}
		if v, ok := translate(lang, inner); ok {
			return []byte(">" + v + "<")
		}
		return m
	})
	return attrText.ReplaceAllFunc(b, func(m []byte) []byte {
		sub := attrText.FindSubmatch(m)
		if v, ok := translate(lang, string(sub[2])); ok {
			return []byte(string(sub[1]) + `="` + v + `"`)
		}
		return m
	})
}
