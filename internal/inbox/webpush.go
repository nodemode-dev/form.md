package inbox

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nodemode-ai/form.md/internal/settings"
)

const recordSize = 4096

var b64 = base64.RawURLEncoding

type vapid struct {
	priv   *ecdsa.PrivateKey
	public string
}

func loadVAPID(ctx context.Context, set *settings.Service) (*vapid, error) {
	raw := set.Get("push.vapid_private")
	if raw == "" {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		der, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return nil, err
		}
		raw = b64.EncodeToString(der)
		if err := set.Set(ctx, map[string]string{"push.vapid_private": raw}); err != nil {
			return nil, err
		}
	}
	der, err := b64.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	parsed, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("push: VAPID key is not ECDSA")
	}
	pub, err := key.PublicKey.ECDH()
	if err != nil {
		return nil, err
	}
	return &vapid{priv: key, public: b64.EncodeToString(pub.Bytes())}, nil
}

func hmacSum(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

func decodeKey(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")
	if b, err := b64.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}

func encrypt(p256dh, auth string, plaintext []byte) ([]byte, error) {
	uaRaw, err := decodeKey(p256dh)
	if err != nil {
		return nil, err
	}
	authSecret, err := decodeKey(auth)
	if err != nil || len(authSecret) < 16 {
		return nil, errors.New("push: bad auth secret")
	}
	ua, err := ecdh.P256().NewPublicKey(uaRaw)
	if err != nil {
		return nil, err
	}
	as, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	shared, err := as.ECDH(ua)
	if err != nil {
		return nil, err
	}
	asPub := as.PublicKey().Bytes()
	prkKey := hmacSum(authSecret, shared)
	info := append(append(append([]byte("WebPush: info\x00"), uaRaw...), asPub...), 1)
	ikm := hmacSum(prkKey, info)
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	prk := hmacSum(salt, ikm)
	cek := hmacSum(prk, []byte("Content-Encoding: aes128gcm\x00\x01"))[:16]
	nonce := hmacSum(prk, []byte("Content-Encoding: nonce\x00\x01"))[:12]
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	body := gcm.Seal(nil, nonce, append(append([]byte{}, plaintext...), 2), nil)
	var out bytes.Buffer
	out.Write(salt)
	_ = binary.Write(&out, binary.BigEndian, uint32(recordSize))
	out.WriteByte(byte(len(asPub)))
	out.Write(asPub)
	out.Write(body)
	return out.Bytes(), nil
}

func subject(set *settings.Service) string {
	if pub := strings.TrimRight(set.Get("instance.public_url"), "/"); strings.HasPrefix(pub, "https://") {
		return pub
	}
	if from := set.Get("mail.from"); strings.Contains(from, "@") {
		if i, j := strings.LastIndex(from, "<"), strings.LastIndex(from, ">"); i >= 0 && j > i {
			from = from[i+1 : j]
		}
		return "mailto:" + strings.TrimSpace(from)
	}
	return "mailto:push@localhost.invalid"
}

func (v *vapid) header(endpoint, sub string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return "", errors.New("push: bad endpoint")
	}
	head := b64.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`))
	claims, _ := json.Marshal(map[string]any{"aud": u.Scheme + "://" + u.Host, "exp": time.Now().Add(12 * time.Hour).Unix(), "sub": sub})
	signing := head + "." + b64.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signing))
	r, s, err := ecdsa.Sign(rand.Reader, v.priv, digest[:])
	if err != nil {
		return "", err
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return fmt.Sprintf("vapid t=%s.%s, k=%s", signing, b64.EncodeToString(sig), v.public), nil
}

type pushResult int

const (
	pushOK pushResult = iota
	pushGone
	pushFailed
)

func (v *vapid) send(ctx context.Context, client *http.Client, sub, endpoint, p256dh, auth string, payload []byte) pushResult {
	body, err := encrypt(p256dh, auth, payload)
	if err != nil {
		return pushGone
	}
	authz, err := v.header(endpoint, sub)
	if err != nil {
		return pushGone
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return pushGone
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("TTL", "86400")
	req.Header.Set("Urgency", "normal")
	req.Header.Set("Authorization", authz)
	res, err := client.Do(req)
	if err != nil {
		return pushFailed
	}
	res.Body.Close()
	switch {
	case res.StatusCode >= 200 && res.StatusCode < 300:
		return pushOK
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone:
		return pushGone
	}
	return pushFailed
}
