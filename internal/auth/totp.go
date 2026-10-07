package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"

	"rsc.io/qr"

	"github.com/nodemode-ai/form.md/internal/secrets"
)

const (
	totpStep   = 30
	totpDigits = 6
	totpSkew   = 1
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func NewTOTPSecret() string {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b32.EncodeToString(b)
}

func totpCode(secret []byte, step int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, secret)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", v%1_000_000)
}

func CheckTOTP(secret, code string, lastStep int64, now time.Time) (int64, bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != totpDigits {
		return 0, false
	}
	key, err := b32.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return 0, false
	}
	current := now.Unix() / totpStep
	for d := -totpSkew; d <= totpSkew; d++ {
		step := current + int64(d)
		if step <= lastStep {
			continue
		}
		if secrets.Equal(totpCode(key, step), code) {
			return step, true
		}
	}
	return 0, false
}

func TOTPURL(issuer, account, secret string) string {
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", "6")
	q.Set("period", "30")
	return "otpauth://totp/" + url.PathEscape(issuer+":"+account) + "?" + q.Encode()
}

func QRSVG(text string) (string, error) {
	c, err := qr.Encode(text, qr.M)
	if err != nil {
		return "", err
	}
	const quiet = 4
	size := c.Size + 2*quiet
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges"><rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="`, size, size, size, size)
	for y := 0; y < c.Size; y++ {
		for x := 0; x < c.Size; x++ {
			if c.Black(x, y) {
				fmt.Fprintf(&b, "M%d %dh1v1h-1z", x+quiet, y+quiet)
			}
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String(), nil
}

const recoveryAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func NewRecoveryCodes(n int) []string {
	out := make([]string, n)
	buf := make([]byte, 10)
	for i := range out {
		if _, err := rand.Read(buf); err != nil {
			panic(err)
		}
		var s strings.Builder
		for j, c := range buf {
			if j == 5 {
				s.WriteByte('-')
			}
			s.WriteByte(recoveryAlphabet[int(c)%len(recoveryAlphabet)])
		}
		out[i] = s.String()
	}
	return out
}

func NormalizeRecovery(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, " ", "")
	if len(code) == 10 && !strings.Contains(code, "-") {
		code = code[:5] + "-" + code[5:]
	}
	return code
}
