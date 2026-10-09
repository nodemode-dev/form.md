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

type Stopper interface {
	Stop()
}

type ExtraListener interface {
	Listen() (net.Listener, error)
}

type Drainer interface {
	Drain(ctx context.Context)
}

const (
	shutdownGrace     = 15 * time.Second
	drainGrace        = 4 * time.Second
	readHeaderTimeout = 10 * time.Second
)

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
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          slog.NewLogLogger(d.Log.Handler(), slog.LevelWarn),
	}
	for _, m := range modules {
		if s, ok := m.(Stopper); ok {
			srv.RegisterOnShutdown(s.Stop)
		}
	}

	ln, err := net.Listen("tcp", d.Config.Addr)
	if err != nil {
		return err
	}

	scheme := "http"
	if d.Config.TLS() {
		scheme = "https"
	}
	var challenges *http.Server
	if d.Config.ACME() {
		m := acmeManager(d)
		srv.TLSConfig = m.TLSConfig()
		if challenges, err = serveChallenges(d, m); err != nil {
			ln.Close()
			return err
		}
	}
	for _, m := range modules {
		if x, ok := m.(ExtraListener); ok {
			extra, err := x.Listen()
			if err != nil {
				d.Log.Warn("extra listener", "module", m.ID(), "err", err)
				continue
			}
			go func() {
				if err := srv.Serve(extra); err != nil && !errors.Is(err, http.ErrServerClosed) {
					d.Log.Warn("extra listener stopped", "module", m.ID(), "err", err)
				}
			}()
		}
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
	err = srv.Shutdown(sctx)
	if challenges != nil {
		challenges.Shutdown(sctx)
	}
	dctx, dcancel := context.WithTimeout(context.Background(), drainGrace)
	defer dcancel()
	for _, m := range modules {
		if d, ok := m.(Drainer); ok {
			d.Drain(dctx)
		}
	}
	return err
}
