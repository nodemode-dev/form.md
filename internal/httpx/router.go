package httpx

import (
	"log/slog"
	"net/http"
)

type Middleware func(http.Handler) http.Handler

type Router struct {
	mux *http.ServeMux
	mw  []Middleware
	log *slog.Logger
}

func NewRouter(log *slog.Logger) *Router {
	r := &Router{mux: http.NewServeMux(), log: log}
	r.Use(RequestID, Recover(log), AccessLog(log), SecurityHeaders)
	r.mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		Error(w, http.StatusNotFound, "not_found", "no such endpoint")
	})
	return r
}

func (r *Router) Use(m ...Middleware) {
	r.mw = append(r.mw, m...)
}

func (r *Router) Handle(pattern string, h http.Handler) {
	r.mux.Handle(pattern, h)
}

func (r *Router) HandleFunc(pattern string, h http.HandlerFunc) {
	r.mux.HandleFunc(pattern, h)
}

func (r *Router) Handler() http.Handler {
	var h http.Handler = r.mux
	for i := len(r.mw) - 1; i >= 0; i-- {
		h = r.mw[i](h)
	}
	return h
}
