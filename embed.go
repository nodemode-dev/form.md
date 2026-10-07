package formmd

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var dist embed.FS

//go:embed NOTICES.md
var notices string

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
