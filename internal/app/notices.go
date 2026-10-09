package app

import (
	"io"
	"net/http"

	"github.com/nodemode-ai/form.md/internal/httpx"
)

type notices struct{}

func Notices() Module { return notices{} }

func (notices) ID() string { return "notices" }

func (notices) Mount(r *httpx.Router, d *Deps) error {
	serve := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = io.WriteString(w, d.Notices)
	}
	r.HandleFunc("GET /notices", serve)
	r.HandleFunc("GET /licenses", serve)
	return nil
}
