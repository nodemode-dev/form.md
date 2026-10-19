package search

import (
	"context"
	"strconv"
	"strings"
)

const (
	maxPDFBytes = 16 << 20
	maxPDFFiles = 4
	maxPDFText  = 200 << 10
)

func (s *Service) SetPDF(extract func(data []byte) ([]string, error), read func(ctx context.Context, id string, max int64) ([]byte, bool)) {
	s.pdfText, s.pdfRead = extract, read
}

func (s *Service) pdfParts(ctx context.Context, pageID string) []passage {
	if s.pdfText == nil || s.pdfRead == nil {
		return nil
	}
	rows, err := s.st.R.QueryContext(ctx, `SELECT f.id, f.name FROM files f JOIN doc_refs r ON r.kind = 'file' AND r.target = f.id
		WHERE r.page_id = ? AND f.mime = 'application/pdf' AND f.size <= ? ORDER BY f.created_at, f.id LIMIT ?`, pageID, maxPDFBytes, maxPDFFiles)
	if err != nil {
		return nil
	}
	type pdf struct{ id, name string }
	var list []pdf
	for rows.Next() {
		var f pdf
		if rows.Scan(&f.id, &f.name) == nil {
			list = append(list, f)
		}
	}
	rows.Close()
	var out []passage
	budget := maxPDFText
	for _, f := range list {
		if budget <= 0 || ctx.Err() != nil {
			break
		}
		data, ok := s.pdfRead(ctx, f.id, maxPDFBytes)
		if !ok {
			continue
		}
		pages, err := s.pdfText(data)
		if err != nil {
			s.log.Debug("search: pdf text", "file", f.id, "err", err)
			continue
		}
		for i, text := range pages {
			if budget <= 0 {
				break
			}
			heading := f.name
			if len(pages) > 1 {
				heading += " > Page " + strconv.Itoa(i+1)
			}
			for _, chunk := range pdfChunks(text, &budget) {
				out = append(out, passage{heading: heading, text: chunk})
			}
		}
	}
	return out
}

func pdfChunks(text string, budget *int) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, w := range strings.Fields(text) {
		if *budget <= 0 {
			break
		}
		if len(w) > maxPassage {
			continue
		}
		if cur.Len() > 0 && cur.Len()+1+len(w) > maxPassage {
			flush()
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
			*budget--
		}
		cur.WriteString(w)
		*budget -= len(w)
	}
	flush()
	return out
}
