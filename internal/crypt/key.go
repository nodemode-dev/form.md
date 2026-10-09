package crypt

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MetaFile = "encryption.json"

const CredentialName = "form-md.key"

var ErrLocked = errors.New("this data directory is encrypted: set FORMMD_KEY_FILE to the key file")

var ErrWrongKey = errors.New("the key does not match this data directory")

type Key struct {
	master []byte
}

type meta struct {
	Version   int    `json:"version"`
	Database  string `json:"database"`
	Files     string `json:"files"`
	Check     string `json:"check"`
	CreatedAt int64  `json:"created_at"`
}

func Generate() string {
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		panic(err)
	}
	return hex.EncodeToString(k)
}

func Parse(raw []byte) (*Key, error) {
	s := strings.TrimSpace(string(raw))
	if b, err := hex.DecodeString(s); err == nil && len(b) == 32 {
		return &Key{master: b}, nil
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil && len(b) == 32 {
			return &Key{master: b}, nil
		}
	}
	return nil, errors.New("the key file must hold 32 bytes as 64 hex characters or base64 (form-md keygen makes one)")
}

func KeyPath(configured string) string {
	if configured != "" {
		return configured
	}
	if dir := os.Getenv("CREDENTIALS_DIRECTORY"); dir != "" {
		p := filepath.Join(dir, CredentialName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func ReadKey(path string) (*Key, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("key file: %w", err)
	}
	return Parse(raw)
}

func Encrypted(dataDir string) bool {
	_, err := os.Stat(filepath.Join(dataDir, MetaFile))
	return err == nil
}

func Load(dataDir, keyFile string, hasData bool) (*Key, error) {
	metaPath := filepath.Join(dataDir, MetaFile)
	raw, err := os.ReadFile(metaPath)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if keyFile == "" {
		if exists {
			return nil, ErrLocked
		}
		return nil, nil
	}
	k, err := ReadKey(keyFile)
	if err != nil {
		return nil, err
	}
	if exists {
		var m meta
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("%s: %w", metaPath, err)
		}
		if !hmac.Equal([]byte(m.Check), []byte(k.check())) {
			return nil, ErrWrongKey
		}
		return k, nil
	}
	if hasData {
		return nil, errors.New("a key is set but this data directory is not encrypted: stop the server and run form-md encrypt first")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	if err := k.WriteMeta(dataDir); err != nil {
		return nil, err
	}
	return k, nil
}

func (k *Key) WriteMeta(dataDir string) error {
	m := meta{Version: 1, Database: "xts-aes-256 per page", Files: "aes-256-gcm in 64 KiB chunks", Check: k.check(), CreatedAt: time.Now().Unix()}
	b, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(filepath.Join(dataDir, MetaFile), append(b, '\n'), 0o600)
}

func (k *Key) check() string {
	m := hmac.New(sha256.New, k.Sub("check", 32))
	m.Write([]byte("form.md key check"))
	return hex.EncodeToString(m.Sum(nil))
}

func (k *Key) Sub(label string, n int) []byte {
	out, err := hkdf.Key(sha256.New, k.master, nil, "form.md/"+label, n)
	if err != nil {
		panic(err)
	}
	return out
}
