package doc

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

type boardItem struct {
	ID   string  `json:"id"`
	K    string  `json:"k"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	W    float64 `json:"w"`
	H    float64 `json:"h"`
	Src  string  `json:"src,omitempty"`
	Text string  `json:"text,omitempty"`
	Href string  `json:"href,omitempty"`
}

type boardSpec struct {
	Height int         `json:"height"`
	Items  []boardItem `json:"items"`
}

const (
	boardWidth     = 1000
	boardMaxItems  = 200
	boardMinHeight = 200
	boardMaxHeight = 8000
)

func boardHeight(h int) int {
	if h < boardMinHeight {
		return 600
	}
	return min(h, boardMaxHeight)
}

func boardLink(h string) string {
	u, err := url.Parse(strings.TrimSpace(h))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		return ""
	}
	return u.String()
}

func parseBoard(raw string) []boardItem {
	var in []boardItem
	if raw == "" || len(raw) > 256<<10 || json.Unmarshal([]byte(raw), &in) != nil {
		return nil
	}
	out := in[:0]
	for _, it := range in {
		if len(out) == boardMaxItems {
			break
		}
		if it.W < 20 || it.H < 20 || it.X < 0 || it.Y < 0 || it.X+it.W > boardWidth+1 || it.Y+it.H > boardMaxHeight+1 {
			continue
		}
		switch it.K {
		case "image":
			if fileID(it.Src) == "" {
				continue
			}
		case "note":
			if len(it.Text) > 4000 {
				it.Text = it.Text[:4000]
			}
		case "link":
			if it.Href = boardLink(it.Href); it.Href == "" {
				continue
			}
		default:
			continue
		}
		out = append(out, it)
	}
	return out
}

func boardEncode(height int, items string) string {
	b, _ := json.Marshal(boardSpec{Height: boardHeight(height), Items: parseBoard(items)})
	return base64.RawURLEncoding.EncodeToString(b)
}

func boardNode(enc string) *pmNode {
	raw, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil || len(raw) > 256<<10 {
		return nil
	}
	var spec boardSpec
	if json.Unmarshal(raw, &spec) != nil {
		return nil
	}
	raw, _ = json.Marshal(spec.Items)
	clean := parseBoard(string(raw))
	if clean == nil {
		clean = []boardItem{}
	}
	raw, _ = json.Marshal(clean)
	n := newNode("board")
	n.attrs["height"] = boardHeight(spec.Height)
	n.attrs["items"] = string(raw)
	return n
}

func boardSVG(height int, items []boardItem, src func(string) string) string {
	var b strings.Builder
	h := strconv.Itoa(boardHeight(height))
	b.WriteString(`<svg class="board__canvas" viewBox="0 0 ` + strconv.Itoa(boardWidth) + " " + h + `" preserveAspectRatio="xMinYMin meet" role="group" aria-label="Board">`)
	for _, it := range items {
		b.WriteString(`<foreignObject class="board__slot" x="` + num(it.X) + `" y="` + num(it.Y) + `" width="` + num(it.W) + `" height="` + num(it.H) + `">`)
		switch it.K {
		case "image":
			b.WriteString(`<div class="board__card board__card--image"><img src="` + esc(src(it.Src)) + `" alt="" loading="lazy" decoding="async" draggable="false"></div>`)
		case "note":
			b.WriteString(`<div class="board__card board__card--note"><div class="board__text">` + esc(it.Text) + `</div></div>`)
		case "link":
			host := it.Href
			if u, err := url.Parse(it.Href); err == nil {
				host = strings.TrimPrefix(u.Host, "www.")
			}
			label := it.Text
			if label == "" {
				label = it.Href
			}
			b.WriteString(`<a class="board__card board__card--link" href="` + esc(it.Href) + `" target="_blank" rel="noopener noreferrer nofollow"><span class="board__host">` + esc(host) + `</span><span class="board__text">` + esc(label) + `</span></a>`)
		}
		b.WriteString(`</foreignObject>`)
	}
	b.WriteString(`</svg>`)
	return b.String()
}
