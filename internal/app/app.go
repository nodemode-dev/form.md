package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/nodemode-ai/form.md/internal/config"
	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/store"
)

type Deps struct {
	Config     config.Config
	Store      *store.Store
	Log        *slog.Logger
	Dist       fs.FS
	Notices    string
	InstanceID string
	Middleware []httpx.Middleware
}

type Module interface {
	ID() string
	Mount(r *httpx.Router, d *Deps) error
}

const shutdownGrace = 15 * time.Second

func Run(ctx context.Context, d *Deps, modules []Module) error {
	r := httpx.NewRouter(d.Log)
	r.Use(d.Middleware...)
	seen := map[string]bool{}
	for _, m := range modules {
		if seen[m.ID()] {
			return fmt.Errorf("module %q registered twice", m.ID())
		}
		seen[m.ID()] = true
		if err := m.Mount(r, d); err != nil {
			return fmt.Errorf("module %q: %w", m.ID(), err)
		}
	}

	srv := &http.Server{
		Handler:           r.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          slog.NewLogLogger(d.Log.Handler(), slog.LevelWarn),
	}

	ln, err := net.Listen("tcp", d.Config.Addr)
	if err != nil {
		return err
	}

	scheme := "http"
	if d.Config.TLS() {
		scheme = "https"
	}
	served := make(chan error, 1)
	go func() {
		if d.Config.TLS() {
			served <- srv.ServeTLS(ln, d.Config.TLSCert, d.Config.TLSKey)
			return
		}
		served <- srv.Serve(ln)
	}()
	d.Log.Info(Product+" listening",
		"url", scheme+"://"+ln.Addr().String(),
		"version", Version,
		"data", d.Config.DataDir)

	select {
	case err := <-served:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	d.Log.Info("shutting down", "grace", shutdownGrace.String())
	sctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	return srv.Shutdown(sctx)
}
