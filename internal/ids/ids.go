package ids

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

var enc = base32.StdEncoding.WithPadding(base32.NoPadding)

func New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return strings.ToLower(enc.EncodeToString(b[:]))
}
