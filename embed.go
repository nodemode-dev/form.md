package formmd

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var dist embed.FS

//go:embed NOTICES.md
var notices string

//go:embed web/src/i18n
var catalogs embed.FS

//go:embed all:.claude/skills/contextual-compression
var contextSkill embed.FS

func ContextSkill() fs.FS {
	sub, err := fs.Sub(contextSkill, ".claude/skills/contextual-compression")
	if err != nil {
		panic(err)
	}
	return sub
}

func Catalogs() fs.FS {
	sub, err := fs.Sub(catalogs, "web/src/i18n")
	if err != nil {
		panic(err)
	}
	return sub
}

func Dist() fs.FS {
	sub, err := fs.Sub(dist, "web/dist")
	if err != nil {
		panic(err)
	}
	return sub
}

func Notices() string {
	return notices
}
