package contextpack

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func (p *Policy) Words(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' }) {
		f = strings.Trim(f, "-_")
		if f == "" || (utf8.RuneCountInString(f) < 2 && !unicode.IsDigit([]rune(f)[0])) || p.stop[f] || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	return out
}

func stem(w string) string {
	for _, suf := range []string{"ings", "ing", "ies", "ied", "es", "ed", "s"} {
		if len(w) > len(suf)+3 && strings.HasSuffix(w, suf) {
			return strings.TrimSuffix(w, suf)
		}
	}
	return w
}

func (p *Policy) bag(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range p.Words(s) {
		out[stem(w)] = true
	}
	return out
}

func (p *Policy) Similarity(a, b string) float64 {
	x, y := p.bag(a), p.bag(b)
	if len(x) == 0 || len(y) == 0 {
		return 0
	}
	both := 0
	for w := range x {
		if y[w] {
			both++
		}
	}
	return float64(both) / float64(len(x)+len(y)-both)
}

func (p *Policy) Overlap(query map[string]bool, text string) float64 {
	if len(query) == 0 {
		return 0
	}
	t := p.bag(text)
	hit := 0
	for w := range query {
		if t[w] {
			hit++
		}
	}
	return float64(hit) / float64(len(query))
}

func sentences(s string, n int) (string, bool) {
	s = strings.Join(strings.Fields(s), " ")
	count := 0
	for i, r := range s {
		if r == '.' || r == '!' || r == '?' {
			if i+1 == len(s) || s[i+1] == ' ' {
				count++
				if count == n {
					return s[:i+1], i+1 < len(s)
				}
			}
		}
	}
	return s, false
}

func clip(s string, max int) (string, bool) {
	if max <= 0 || len(s) <= max {
		return s, false
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	if i := strings.LastIndexByte(s[:cut], ' '); i > max/2 {
		cut = i
	}
	return strings.TrimRight(s[:cut], " ,;:"), true
}

func (p *Policy) Digest(s string) string {
	out, more := sentences(s, p.Compression.Digest.Sentences)
	out, cut := clip(out, p.Compression.Digest.MaxChars)
	if more || cut {
		out += " ..."
	}
	return out
}

func (p *Policy) Outline(markdown string) (string, int) {
	var b strings.Builder
	shown, total := 0, 0
	inCode := false
	for _, line := range strings.Split(markdown, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode = !inCode
			continue
		}
		if inCode || !strings.HasPrefix(line, "#") {
			continue
		}
		depth := len(line) - len(strings.TrimLeft(line, "#"))
		title := strings.TrimSpace(line[depth:])
		if depth > 6 || title == "" {
			continue
		}
		total++
		if depth > p.Compression.Outline.MaxDepth || shown >= p.Compression.Outline.MaxHeadings {
			continue
		}
		shown++
		b.WriteString(strings.Repeat("  ", depth-1) + "- " + title + "\n")
	}
	return b.String(), total - shown
}
