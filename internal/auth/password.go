package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
	MinPassword  = 8
	MaxPassword  = 256
)

var hashSlots = make(chan struct{}, 4)

var dummyHash = func() string {
	h, err := hashPassword("form.md timing guard")
	if err != nil {
		panic(err)
	}
	return h
}()

func acquire(ctx context.Context) error {
	select {
	case hashSlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func release() { <-hashSlots }

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

func HashPassword(ctx context.Context, password string) (string, error) {
	if err := acquire(ctx); err != nil {
		return "", err
	}
	defer release()
	return hashPassword(password)
}

func VerifyPassword(ctx context.Context, encoded, password string) (bool, error) {
	if encoded == "" {
		encoded = dummyHash
		password += "\x00"
	}
	if err := acquire(ctx); err != nil {
		return false, err
	}
	defer release()
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("password: unsupported hash format")
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, err
	}
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := enc.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func ValidPassword(password string) bool {
	n := utf8.RuneCountInString(password)
	return n >= MinPassword && n <= MaxPassword
}
