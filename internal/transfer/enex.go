package transfer

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"io"
	"mime"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type enexResource struct {
	Data struct {
		Encoding string `xml:"encoding,attr"`
		Value    string `xml:",chardata"`
	} `xml:"data"`
	Mime  string `xml:"mime"`
	Attrs struct {
		FileName string `xml:"file-name"`
	} `xml:"resource-attributes"`
	Recognition string `xml:"recognition"`
}

type enexNote struct {
	Title     string         `xml:"title"`
	Content   string         `xml:"content"`
	Created   string         `xml:"created"`
	Updated   string         `xml:"updated"`
	Tags      []string       `xml:"tag"`
	Resources []enexResource `xml:"resource"`
	Attrs     struct {
		Author    string `xml:"author"`
		SourceURL string `xml:"source-url"`
	} `xml:"note-attributes"`
}

func importENEX(name string, r io.Reader, b *Bundle) {
	notebook := &Node{Title: strings.TrimSpace(stripExt(name))}
	if notebook.Title == "" {
		notebook.Title = "Evernote"
	}
	dec := xml.NewDecoder(r)
	dec.Strict = false
	dec.Entity = xml.HTMLEntity
	dec.CharsetReader = func(_ string, in io.Reader) (io.Reader, error) { return in, nil }
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "note" {
			continue
		}
		var note enexNote
		if err := dec.DecodeElement(&note, &se); err != nil {
			b.Warn("Skipped a note that could not be read")
			continue
		}
		notebook.Children = append(notebook.Children, enexToNode(&note, b))
	}
	if len(notebook.Children) == 0 {
		b.Warn("No notes were found in " + name)
		return
	}
	b.Roots = append(b.Roots, notebook)
}

func enexToNode(note *enexNote, b *Bundle) *Node {
	n := &Node{Title: strings.TrimSpace(note.Title)}
	if n.Title == "" {
		n.Title = "Untitled note"
	}
	byHash := map[string]*Asset{}
	var order []*Asset
	mimes := map[*Asset]string{}
	for i := range note.Resources {
		res := &note.Resources[i]
		raw := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, res.Data.Value)
		data, err := base64.StdEncoding.DecodeString(raw)
		if err != nil || len(data) == 0 {
			continue
		}
		sum := md5.Sum(data)
		fname := strings.TrimSpace(res.Attrs.FileName)
		if fname == "" {
			fname = "attachment"
			if exts, _ := mime.ExtensionsByType(res.Mime); len(exts) > 0 {
				fname += exts[0]
			}
		}
		a := &Asset{Name: fname, Data: data}
		byHash[hex.EncodeToString(sum[:])] = a
		order = append(order, a)
		mimes[a] = res.Mime
	}
	used := map[*Asset]bool{}
	content := note.Content
	if i := strings.Index(content, "<en-note"); i >= 0 {
		content = content[i:]
	}
	root, err := html.Parse(strings.NewReader(content))
	if err == nil {
		n.Body = htmlToMarkdown(root, htmlOpts{
			enMedia: func(hash, typ string) (string, bool) {
				a, ok := byHash[strings.ToLower(hash)]
				if !ok {
					return "", false
				}
				used[a] = true
				return b.AssetRef(a), strings.HasPrefix(typ, "image/")
			},
			media: func(src, _ string) (string, bool) {
				if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
					return src, true
				}
				return "", false
			},
		}, b.Warn)
	}
	var extra []string
	for _, a := range order {
		if used[a] {
			continue
		}
		dest := b.AssetRef(a)
		if strings.HasPrefix(mimes[a], "image/") {
			extra = append(extra, "!["+escapeText(a.Name)+"]("+dest+")")
		} else {
			extra = append(extra, "["+escapeText(a.Name)+"]("+dest+")")
		}
	}
	var meta []string
	if len(note.Tags) > 0 {
		meta = append(meta, "Tags: "+escapeText(strings.Join(note.Tags, ", ")))
	}
	if t := enexTime(note.Created); !t.IsZero() {
		meta = append(meta, "Created: "+t.Format("January 2, 2006 15:04"))
	}
	if u := strings.TrimSpace(note.Attrs.SourceURL); strings.HasPrefix(u, "http") {
		meta = append(meta, "Source: <"+u+">")
	}
	parts := []string{}
	if strings.TrimSpace(n.Body) != "" {
		parts = append(parts, n.Body)
	}
	parts = append(parts, extra...)
	if len(meta) > 0 {
		parts = append(parts, "---", "*"+strings.Join(meta, " · ")+"*")
	}
	n.Body = strings.Join(parts, "\n\n")
	return n
}

func enexTime(s string) time.Time {
	t, err := time.Parse("20060102T150405Z", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}
	}
	return t
}

func looksLikeENEX(head []byte) bool {
	return bytes.Contains(head, []byte("<en-export")) || bytes.Contains(head, []byte("evernote-export"))
}
