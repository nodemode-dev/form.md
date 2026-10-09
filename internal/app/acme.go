package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"path/filepath"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/nodemode-ai/form.md/internal/crypt"
)

type sealedCache struct {
	dir autocert.DirCache
	key *crypt.Key
}

func (c sealedCache) Get(ctx context.Context, name string) ([]byte, error) {
	raw, err := c.dir.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	out, err := c.key.Unseal(raw)
	if err != nil {
		return nil, autocert.ErrCacheMiss
	}
	return out, nil
}

func (c sealedCache) Put(ctx context.Context, name string, data []byte) error {
	return c.dir.Put(ctx, name, c.key.Seal(data))
}

func (c sealedCache) Delete(ctx context.Context, name string) error {
	return c.dir.Delete(ctx, name)
}

func acmeManager(d *Deps) *autocert.Manager {
	dir := autocert.DirCache(filepath.Join(d.Config.DataDir, "acme"))
	var cache autocert.Cache = dir
	if d.Store != nil && d.Store.Key != nil {
		cache = sealedCache{dir: dir, key: d.Store.Key}
	}
	return &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(d.Config.ACMEDomains...),
		Cache:      cache,
		Email:      d.Config.ACMEEmail,
		Client:     &acme.Client{DirectoryURL: d.Config.ACMEDirectory},
	}
}

func serveChallenges(d *Deps, m *autocert.Manager) (*http.Server, error) {
	ln, err := net.Listen("tcp", d.Config.HTTPAddr)
	if err != nil {
		return nil, err
	}
	hs := &http.Server{Handler: m.HTTPHandler(nil), ReadHeaderTimeout: readHeaderTimeout}
	go func() {
		if err := hs.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			d.Log.Warn("acme http listener", "err", err)
		}
	}()
	d.Log.Info("acme http listener", "addr", ln.Addr().String(), "domains", d.Config.ACMEDomains)
	return hs, nil
}
