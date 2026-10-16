package i18n

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

var catalogs = map[string]map[string]string{"de": de}

func LoadFS(fsys fs.FS) error {
	langs, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return err
	}
	for _, l := range langs {
		if !l.IsDir() {
			continue
		}
		parts, err := fs.Glob(fsys, l.Name()+"/*.json")
		if err != nil {
			return err
		}
		c := catalogs[l.Name()]
		if c == nil {
			c = map[string]string{}
			catalogs[l.Name()] = c
		}
		for _, p := range parts {
			raw, err := fs.ReadFile(fsys, p)
			if err != nil {
				return err
			}
			var m map[string]string
			if err := json.Unmarshal(raw, &m); err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			for k, v := range m {
				if _, ok := c[k]; !ok {
					c[k] = v
				}
			}
		}
	}
	return nil
}

func Has(lang, msg string) (string, bool) {
	v, ok := catalogs[Base(lang)][msg]
	return v, ok
}

var aliases = map[string]string{
	"zh": "zh-Hans", "zh-cn": "zh-Hans", "zh-sg": "zh-Hans", "zh-hans-cn": "zh-Hans",
	"zh-tw": "zh-Hant", "zh-hk": "zh-Hant", "zh-mo": "zh-Hant", "zh-hant-tw": "zh-Hant",
	"pt": "pt-BR", "pt-pt": "pt-BR", "es-419": "es-MX", "es-us": "es-MX", "no": "nb", "nn": "nb", "nb-no": "nb",
}

func Resolve(lang string) (string, bool) {
	tag := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(lang), "_", "-"))
	if tag == "" {
		return "", false
	}
	if tag == "en" || strings.HasPrefix(tag, "en-") {
		return "en", true
	}
	if a, ok := aliases[tag]; ok {
		return a, true
	}
	for name := range catalogs {
		if strings.ToLower(name) == tag {
			return name, true
		}
	}
	base, _, _ := strings.Cut(tag, "-")
	if a, ok := aliases[base]; ok {
		return a, true
	}
	if _, ok := catalogs[base]; ok {
		return base, true
	}
	return "", false
}

func Base(lang string) string {
	if name, ok := Resolve(lang); ok {
		return name
	}
	return "en"
}

func T(lang, msg string, args ...any) string {
	if c, ok := catalogs[Base(lang)]; ok {
		if v, ok := c[msg]; ok {
			msg = v
		}
	}
	if len(args) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, args...)
}

func UserLang(ctx context.Context, db *sql.DB, userID string) string {
	var lang string
	if userID == "" || db.QueryRowContext(ctx, "SELECT lang FROM users WHERE id = ?", userID).Scan(&lang) != nil {
		return "en"
	}
	return Base(lang)
}
