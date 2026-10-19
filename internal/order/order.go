package order

import (
	"errors"
	"strings"
)

const digits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

const base = len(digits)

var ErrOrder = errors.New("order: keys out of order")

func value(s string, i int) int {
	if i >= len(s) {
		return 0
	}
	return strings.IndexByte(digits, s[i])
}

func Valid(key string) bool {
	if key == "" || key[len(key)-1] == '0' {
		return false
	}
	for i := 0; i < len(key); i++ {
		if strings.IndexByte(digits, key[i]) < 0 {
			return false
		}
	}
	return true
}

func Between(a, b string) (string, error) {
	if b != "" && a >= b {
		return "", ErrOrder
	}
	return midpoint(a, b), nil
}

func midpoint(a, b string) string {
	if b == "" {
		d := value(a, 0)
		if a == "" || d < base-1 {
			return string(digits[(d+base+1)/2])
		}
		return a[:1] + midpoint(a[1:], "")
	}
	i := 0
	for value(a, i) == value(b, i) {
		i++
	}
	prefix := b[:i]
	lo, hi := value(a, i), value(b, i)
	if hi-lo > 1 {
		return prefix + string(digits[(lo+hi)/2])
	}
	if len(b) > i+1 {
		return b[:i+1]
	}
	rest := ""
	if len(a) > i+1 {
		rest = a[i+1:]
	}
	return prefix + string(digits[lo]) + midpoint(rest, "")
}

func After(a string) string {
	if next, ok := bump(a); ok {
		return next
	}
	return midpoint(a, "")
}

func bump(a string) (string, bool) {
	for i := len(a) - 1; i >= 0; i-- {
		v := value(a, i)
		if v < 0 {
			return "", false
		}
		if v < base-1 {
			return a[:i] + string(digits[v+1]), true
		}
	}
	if a == "" {
		return "", false
	}
	return a + "1", true
}

func Before(b string) string {
	return midpoint("", b)
}

func Sequence(after string, n int) []string {
	if n <= 0 {
		return nil
	}
	head := After(after)
	width := 1
	for m := n - 1; m >= base; m /= base {
		width++
	}
	out := make([]string, n)
	buf := make([]byte, width+1)
	buf[width] = digits[base/2]
	for i := range out {
		v := i
		for j := width - 1; j >= 0; j-- {
			buf[j] = digits[v%base]
			v /= base
		}
		out[i] = head + string(buf)
	}
	return out
}
