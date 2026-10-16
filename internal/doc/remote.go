package doc

import (
	"context"
	"strings"
	"time"

	"github.com/reearth/ygo/crdt"

	"github.com/nodemode-ai/form.md/internal/policy"
)

type RemoteImporter func(ctx context.Context, p *policy.Principal, pageID, url string) (string, error)

const remoteMax = 20

func (s *Service) SetRemoteImporter(fn RemoteImporter) { s.remote = fn }

func remoteImages(n any, out map[string]bool) {
	for _, c := range children(n) {
		el, ok := c.(*crdt.YXmlElement)
		if !ok {
			continue
		}
		if el.NodeName == "image" {
			if src := attrString(el, "src"); strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://") {
				out[src] = true
			}
		}
		remoteImages(el, out)
	}
}

func swapImages(n any, txn *crdt.Transaction, done map[string]string) {
	for _, c := range children(n) {
		el, ok := c.(*crdt.YXmlElement)
		if !ok {
			continue
		}
		if el.NodeName == "image" {
			if id, ok := done[attrString(el, "src")]; ok {
				el.SetAttributeValue(txn, "src", "f:"+id)
			}
		}
		swapImages(el, txn, done)
	}
}

func (s *Service) importRemote(p *policy.Principal, pageID string) {
	if s.remote == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		urls := map[string]bool{}
		if err := s.withRoom(ctx, pageID, func(r *room) { remoteImages(r.fragment(), urls) }); err != nil || len(urls) == 0 {
			return
		}
		done := map[string]string{}
		for u := range urls {
			if len(done) == remoteMax {
				break
			}
			id, err := s.remote(ctx, p, pageID, u)
			if err != nil {
				s.log.Info("doc: remote picture not imported", "page", pageID, "err", err)
				continue
			}
			done[u] = id
		}
		if len(done) == 0 {
			return
		}
		_ = s.withRoom(ctx, pageID, func(r *room) {
			frag := r.fragment()
			r.doc.Transact(func(txn *crdt.Transaction) { swapImages(frag, txn, done) }, &origin{user: p.UserID})
		})
	}()
}
