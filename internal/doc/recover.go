package doc

import "context"

func (s *Service) FlushAll(ctx context.Context) {
	s.mu.Lock()
	rooms := make([]*room, 0, len(s.rooms))
	for _, r := range s.rooms {
		rooms = append(rooms, r)
	}
	s.mu.Unlock()
	for _, r := range rooms {
		s.materialise(ctx, r)
	}
}

func (s *Service) Recover(ctx context.Context) {
	rows, err := s.st.R.QueryContext(ctx, "SELECT page_id FROM doc_state WHERE updated_at > materialised_at")
	if err != nil {
		s.log.Warn("doc: recover", "err", err)
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		r, err := s.acquire(ctx, id)
		if err != nil {
			continue
		}
		r.mu.Lock()
		r.dirty = true
		r.mu.Unlock()
		s.materialise(ctx, r)
		s.release(r)
	}
	if len(ids) > 0 {
		s.log.Info("doc: brought stored text up to date after an unclean stop", "pages", len(ids))
	}
}
