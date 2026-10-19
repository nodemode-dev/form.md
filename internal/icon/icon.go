package icon

import (
	"regexp"
	"strings"
)

var pattern = regexp.MustCompile(`^(?:i:[a-z0-9]+(?:-[a-z0-9]+)*(?::(?:violet|blue|cyan|teal|green|lime|amber|orange|rose|pink))?|f:[a-z2-7]{26})$`)

var names = map[string]bool{}

func init() {
	for _, n := range strings.Fields(`page note notebook book bookmark pen pencil clipboard list checklist quote inbox archive folder paperclip tag
		calendar clock hourglass alarm briefcase chart-bar chart-line chart-pie target flag rocket lightbulb puzzle layers kanban
		code terminal bug database server cloud gear wrench user users chat mail phone megaphone heart star home building map pin
		globe compass leaf tree flower sun moon mountain flame drop snowflake wave camera image music film gift coffee cart wallet
		coin key lock shield trophy crown anchor bell link bolt atom flask cap palette brush dice plane car paw fish apple
		eye box grid table scale cross`) {
		names[n] = true
	}
}

func Normalize(v string) string {
	if rest, ok := strings.CutPrefix(v, "i:sparkle"); ok && (rest == "" || rest[0] == ':') {
		return "i:bolt" + rest
	}
	return v
}

func Valid(v string) bool {
	if v == "" {
		return true
	}
	if !pattern.MatchString(v) {
		return false
	}
	if rest, ok := strings.CutPrefix(v, "i:"); ok {
		name, _, _ := strings.Cut(rest, ":")
		return names[name]
	}
	return true
}
