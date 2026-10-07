package icon

import "regexp"

var pattern = regexp.MustCompile(`^(?:i:[a-z0-9]+(?:-[a-z0-9]+)*(?::(?:violet|blue|cyan|teal|green|lime|amber|orange|rose|pink))?|f:[a-z2-7]{26})$`)

func Valid(v string) bool {
	return v == "" || pattern.MatchString(v)
}
