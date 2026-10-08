package search

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxPassage = 1500

type passage struct {
	heading string
	text    string
}

var (
	reFence   = regexp.MustCompile("^\\s*(```|~~~)")
	reHeading = regexp.MustCompile(`^(#{1,6})\s+(.*?)\s*#*\s*$`)
	reImage   = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	reLink    = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)
	reComment = regexp.MustCompile(`<!--.*?-->`)
	reTag     = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	reLead    = regexp.MustCompile(`^\s*(?:[-*+]\s+\[[ xX]\]\s+|[-*+]\s+|\d+[.)]\s+|>\s?)+`)
	reAlert   = regexp.MustCompile(`^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*`)
	reMarks   = regexp.MustCompile("[*_~`]{1,3}")
	reRule    = regexp.MustCompile(`^\s*([-*_=|:]\s*){3,}$`)
	reSpace   = regexp.MustCompile(`\s+`)
)

func cleanLine(line string) string {
	if reRule.MatchString(line) {
		return ""
	}
	s := reComment.ReplaceAllString(line, " ")
	s = reImage.ReplaceAllString(s, "$1")
	s = reLink.ReplaceAllStringFunc(s, func(m string) string {
		sub := reLink.FindStringSubmatch(m)
		if strings.HasPrefix(sub[2], "/p/") || strings.HasPrefix(sub[2], "/files/") {
			return sub[1]
		}
		return sub[1] + " " + sub[2]
	})
	s = reTag.ReplaceAllString(s, " ")
	s = reLead.ReplaceAllString(s, "")
	s = reAlert.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "|", " ")
	s = reMarks.ReplaceAllString(s, "")
	return strings.TrimSpace(reSpace.ReplaceAllString(s, " "))
}

func passages(md string) []passage {
	var out []passage
	var levels [7]string
	heading := ""
	pending := false
	var cur strings.Builder
	flush := func() {
		t := strings.TrimSpace(cur.String())
		cur.Reset()
		if t == "" {
			if pending {
				out = append(out, passage{heading: heading})
			}
			pending = false
			return
		}
		pending = false
		for t != "" {
			chunk := t
			if len(chunk) > maxPassage {
				cut := strings.LastIndexAny(chunk[:maxPassage], " \n")
				if cut < maxPassage/2 {
					cut = maxPassage
					for cut > 0 && !utf8.RuneStart(chunk[cut]) {
						cut--
					}
				}
				chunk = chunk[:cut]
			}
			out = append(out, passage{heading: heading, text: strings.TrimSpace(chunk)})
			t = strings.TrimSpace(t[len(chunk):])
		}
	}
	fence := false
	for _, line := range strings.Split(md, "\n") {
		if reFence.MatchString(line) {
			fence = !fence
			continue
		}
		if !fence {
			if m := reHeading.FindStringSubmatch(line); m != nil {
				flush()
				lvl := len(m[1])
				levels[lvl] = cleanLine(m[2])
				for i := lvl + 1; i < len(levels); i++ {
					levels[i] = ""
				}
				var path []string
				for i := 1; i <= lvl; i++ {
					if levels[i] != "" {
						path = append(path, levels[i])
					}
				}
				heading = strings.Join(path, " > ")
				pending = true
				continue
			}
		}
		text := strings.TrimSpace(line)
		if !fence {
			text = cleanLine(line)
		}
		if text != "" {
			cur.WriteString(text)
			cur.WriteByte('\n')
		}
	}
	flush()
	return out
}
