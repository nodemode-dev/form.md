package search

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	markOpen  = "\x02"
	markClose = "\x03"
	window    = 18
)

var latin = map[rune]string{
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'ç': "c", 'ć': "c", 'ĉ': "c", 'ċ': "c", 'č': "c", 'ď': "d", 'đ': "d",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ĕ': "e", 'ė': "e", 'ę': "e", 'ě': "e",
	'ĝ': "g", 'ğ': "g", 'ġ': "g", 'ģ': "g", 'ĥ': "h", 'ħ': "h",
	'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ĩ': "i", 'ī': "i", 'ĭ': "i", 'į': "i", 'ı': "i", 'ĵ': "j", 'ķ': "k",
	'ĺ': "l", 'ļ': "l", 'ľ': "l", 'ŀ': "l", 'ł': "l", 'ñ': "n", 'ń': "n", 'ņ': "n", 'ň': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o", 'ŏ': "o", 'ő': "o",
	'ŕ': "r", 'ŗ': "r", 'ř': "r", 'ś': "s", 'ŝ': "s", 'ş': "s", 'š': "s", 'ß': "ss", 'ţ': "t", 'ť': "t", 'ŧ': "t",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ũ': "u", 'ū': "u", 'ŭ': "u", 'ů': "u", 'ű': "u", 'ų': "u",
	'ŵ': "w", 'ý': "y", 'ÿ': "y", 'ŷ': "y", 'ź': "z", 'ż': "z", 'ž': "z", 'æ': "ae", 'œ': "oe",
}

var suffixes = []string{"ations", "ation", "ings", "ing", "edly", "ies", "ied", "ed", "es", "s", "ly", "ments", "ment", "ness"}

func fold(w string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(w) {
		if s, ok := latin[r]; ok {
			b.WriteString(s)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func stem(w string) string {
	for _, s := range suffixes {
		if len(w)-len(s) >= 3 && strings.HasSuffix(w, s) {
			return w[:len(w)-len(s)]
		}
	}
	return w
}

func queryTerms(q string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(q, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if t := stem(fold(f)); t != "" {
			out = append(out, t)
		}
	}
	return out
}

type token struct {
	start, end int
	hit        bool
}

func tokens(text string, terms []string) []token {
	var out []token
	i := 0
	for i < len(text) {
		r, n := utf8.DecodeRuneInString(text[i:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			i += n
			continue
		}
		j := i
		for j < len(text) {
			r2, n2 := utf8.DecodeRuneInString(text[j:])
			if !unicode.IsLetter(r2) && !unicode.IsDigit(r2) {
				break
			}
			j += n2
		}
		w := fold(text[i:j])
		hit := false
		for _, t := range terms {
			if strings.HasPrefix(w, t) || strings.HasPrefix(stem(w), t) || (len(t) > 3 && strings.HasPrefix(t, stem(w)) && len(stem(w)) >= 3) {
				hit = true
				break
			}
		}
		out = append(out, token{i, j, hit})
		i = j
	}
	return out
}

func mark(text string, terms []string) (string, bool) {
	toks := tokens(text, terms)
	var b strings.Builder
	last, any := 0, false
	for _, t := range toks {
		if !t.hit {
			continue
		}
		any = true
		b.WriteString(text[last:t.start])
		b.WriteString(markOpen + text[t.start:t.end] + markClose)
		last = t.end
	}
	b.WriteString(text[last:])
	return b.String(), any
}

func snippet(text string, terms []string) (string, bool) {
	text = strings.Join(strings.Fields(text), " ")
	toks := tokens(text, terms)
	if len(toks) == 0 {
		return "", false
	}
	best, bestHits := 0, -1
	for s := 0; s < len(toks); s++ {
		n := 0
		for k := s; k < len(toks) && k < s+window; k++ {
			if toks[k].hit {
				n++
			}
		}
		if n > bestHits {
			best, bestHits = s, n
		}
		if s+window >= len(toks) {
			break
		}
	}
	end := min(best+window, len(toks))
	from, to := toks[best].start, toks[end-1].end
	if best == 0 {
		from = 0
	}
	if end == len(toks) {
		to = len(text)
	}
	out, any := mark(text[from:to], terms)
	if best > 0 {
		out = "..." + out
	}
	if end < len(toks) {
		out += "..."
	}
	return out, any
}
