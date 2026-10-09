package static

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var assetRef = regexp.MustCompile(`/assets/[A-Za-z0-9._-]+`)

type offlineManifest struct {
	Version string   `json:"version"`
	Core    []string `json:"core"`
	Assets  []string `json:"assets"`
}

var (
	offlineOnce sync.Once
	offlineBody []byte
)

func (h *Handler) offline() []byte {
	offlineOnce.Do(func() {
		m := offlineManifest{Core: []string{}, Assets: []string{}}
		core := map[string]bool{}
		if h.index != nil {
			m.Version = h.index.etag
			for _, ref := range assetRef.FindAll(h.index.body, -1) {
				core[string(ref)] = true
			}
		}
		for _, list := range h.preloads {
			for _, href := range list {
				core[href] = true
			}
		}
		for p := range h.files {
			if strings.HasPrefix(p, "/assets/") {
				m.Assets = append(m.Assets, p)
			}
		}
		for p := range core {
			if _, ok := h.files[p]; ok {
				m.Core = append(m.Core, p)
			}
		}
		sort.Strings(m.Core)
		sort.Strings(m.Assets)
		offlineBody, _ = json.Marshal(m)
	})
	return offlineBody
}

func (h *Handler) serveOffline(w http.ResponseWriter, r *http.Request) {
	a := &asset{body: h.offline(), ctype: "application/json"}
	a.etag = etag(a.body)
	h.serve(w, r, a, false)
}
