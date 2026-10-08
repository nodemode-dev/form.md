package doc

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"math"
	"strings"

	"github.com/reearth/ygo/crdt"
)

const Root = "prosemirror"

var markOrder = []string{"link", "strong", "em", "strike", "underline", "highlight", "color", "code"}

var textblocks = map[string]bool{"paragraph": true, "heading": true, "code_block": true, "toggle_title": true}

var atoms = map[string]bool{
	"horizontal_rule": true, "image": true, "video": true, "audio": true, "file": true,
	"bookmark": true, "embed": true, "toc": true, "database": true, "breadcrumb": true, "math_block": true, "synced": true,
}

var blockIDs = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

func NewBlockID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return blockIDs.EncodeToString(b[:])
}

func markName(key string) string {
	if i := strings.Index(key, "--"); i > 0 {
		return key[:i]
	}
	return key
}

func attrString(el *crdt.YXmlElement, key string) string {
	v, ok := el.GetAttributeValue(key)
	if !ok || v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == math.Trunc(x) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	case int, int64, uint64, bool:
		return fmt.Sprint(x)
	}
	return ""
}

func attrInt(el *crdt.YXmlElement, key string, def int) int {
	v, ok := el.GetAttributeValue(key)
	if !ok || v == nil {
		return def
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	case uint64:
		return int(x)
	case string:
		var n int
		if _, err := fmt.Sscanf(x, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func attrBool(el *crdt.YXmlElement, key string) bool {
	v, ok := el.GetAttributeValue(key)
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x == "true"
	}
	return false
}

func markAttr(v any, key string) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func fileID(ref string) string {
	if id, ok := strings.CutPrefix(ref, "f:"); ok {
		return id
	}
	if id, ok := strings.CutPrefix(ref, "/files/"); ok {
		return id
	}
	return ""
}

func fileURL(ref string) string {
	if id := fileID(ref); id != "" {
		return "/files/" + id
	}
	return ref
}

func children(n any) []any {
	var kids []any
	switch x := n.(type) {
	case *crdt.YXmlFragment:
		for _, c := range x.Children() {
			kids = append(kids, c)
		}
	case *crdt.YXmlElement:
		for _, c := range x.Children() {
			kids = append(kids, c)
		}
	}
	return kids
}
