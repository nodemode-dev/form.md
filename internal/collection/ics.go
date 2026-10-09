package collection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/httpx"
	"github.com/nodemode-ai/form.md/internal/policy"
	"github.com/nodemode-ai/form.md/internal/store"
)

const maxFeedEvents = 2000

func (s *Service) CalendarFeed(ctx context.Context, p *policy.Principal, viewID string, reset bool) (string, error) {
	v, err := s.view(ctx, viewID)
	if err != nil {
		return "", err
	}
	if _, _, err := s.readable(ctx, p, v.CollectionID); err != nil {
		return "", err
	}
	if !p.IsSession() {
		return "", httpx.ErrSessionOnly
	}
	if reset {
		if _, err := s.st.W.ExecContext(ctx, "DELETE FROM calendar_feeds WHERE view_id = ? AND user_id = ?", viewID, p.UserID); err != nil {
			return "", err
		}
	}
	var token string
	err = s.st.R.QueryRowContext(ctx, "SELECT token FROM calendar_feeds WHERE view_id = ? AND user_id = ?", viewID, p.UserID).Scan(&token)
	if err == nil {
		return token, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	token = newToken()
	if _, err := s.st.W.ExecContext(ctx, "INSERT INTO calendar_feeds (token, view_id, user_id, created_at) VALUES (?, ?, ?, ?)", token, viewID, p.UserID, store.NowMS()); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) FeedOwner(ctx context.Context, token string) (string, string, error) {
	var viewID, userID string
	err := s.st.R.QueryRowContext(ctx, "SELECT view_id, user_id FROM calendar_feeds WHERE token = ?", token).Scan(&viewID, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", httpx.ErrNotFound
	}
	return viewID, userID, err
}

func icsText(s string) string {
	r := strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`)
	return r.Replace(s)
}

func icsFold(line string) string {
	if len(line) <= 74 {
		return line + "\r\n"
	}
	var b strings.Builder
	n := 0
	for _, r := range line {
		size := len(string(r))
		if n+size > 74 {
			b.WriteString("\r\n ")
			n = 1
		}
		b.WriteRune(r)
		n += size
	}
	b.WriteString("\r\n")
	return b.String()
}

func (s *Service) ICS(ctx context.Context, p *policy.Principal, viewID, base, host string) ([]byte, string, error) {
	v, err := s.view(ctx, viewID)
	if err != nil {
		return nil, "", err
	}
	m, _, err := s.readable(ctx, p, v.CollectionID)
	if err != nil {
		return nil, "", err
	}
	ev := s.evaluator(ctx, p)
	defer ev.close()
	sn := ev.snap(v.CollectionID)
	if sn == nil {
		return nil, "", store.ErrNotFound
	}
	propID, _ := v.Config.Layout["date"].(string)
	var dp *Property
	if propID != "" {
		dp = sn.schema.ByID(propID)
	}
	if dp == nil || dp.Type != TDate {
		dp = nil
		for _, x := range sn.schema {
			if x.Type == TDate {
				dp = x
				break
			}
		}
	}
	name := m.Title
	if name == "" {
		name = "form.md"
	}
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//form.md//calendar//EN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\n")
	b.WriteString(icsFold("X-WR-CALNAME:" + icsText(name+" · "+v.Name)))
	stamp := time.Now().UTC().Format("20060102T150405Z")
	if dp != nil {
		count := 0
		for _, r := range s.results.ordered(ev, sn, ViewConfig{Filter: v.Config.Filter}, QueryInput{}) {
			raw, ok := r.Vals[dp.ID]
			if !ok {
				continue
			}
			var d DateVal
			if json.Unmarshal(raw, &d) != nil || d.Start == "" {
				continue
			}
			start, timed, ok := parseDay(d.Start)
			if !ok {
				continue
			}
			end := start
			if d.End != "" {
				if e, _, ok := parseDay(d.End); ok {
					end = e
				}
			}
			b.WriteString("BEGIN:VEVENT\r\n")
			b.WriteString("UID:" + r.ID + "@" + host + "\r\n")
			b.WriteString("DTSTAMP:" + stamp + "\r\n")
			if timed {
				if !end.After(start) {
					end = start.Add(time.Hour)
				}
				b.WriteString("DTSTART:" + start.Format("20060102T150405") + "\r\n")
				b.WriteString("DTEND:" + end.Format("20060102T150405") + "\r\n")
			} else {
				b.WriteString("DTSTART;VALUE=DATE:" + start.Format("20060102") + "\r\n")
				b.WriteString("DTEND;VALUE=DATE:" + end.AddDate(0, 0, 1).Format("20060102") + "\r\n")
			}
			title := r.Title
			if title == "" {
				title = "Untitled"
			}
			b.WriteString(icsFold("SUMMARY:" + icsText(title)))
			b.WriteString(icsFold("URL:" + base + "/p/" + r.ID))
			b.WriteString("LAST-MODIFIED:" + time.UnixMilli(r.UpdatedAt).UTC().Format("20060102T150405Z") + "\r\n")
			b.WriteString("END:VEVENT\r\n")
			count++
			if count >= maxFeedEvents {
				break
			}
		}
	}
	b.WriteString("END:VCALENDAR\r\n")
	return []byte(b.String()), name, nil
}
