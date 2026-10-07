package app

import (
	"context"
	"net/http"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
)

type health struct{}

func Health() Module { return health{} }

func (health) ID() string { return "health" }

type healthBody struct {
	Status  string `json:"status"`
	Product string `json:"product"`
	Version string `json:"version"`
	Storage string `json:"storage"`
}

func (health) Mount(r *httpx.Router, d *Deps) error {
	r.HandleFunc("GET /healthz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		body := healthBody{Status: "ok", Product: Product, Version: Version, Storage: "ok"}
		status := http.StatusOK
		if err := d.Store.Ping(ctx); err != nil {
			d.Log.Warn("health: storage ping failed", "err", err)
			body.Status, body.Storage, status = "degraded", "error", http.StatusServiceUnavailable
		}
		httpx.JSON(w, status, body)
	})
	return nil
}
