package config

import (
	"strings"
	"testing"
)

const goodKey = "0123456789abcdef0123456789abcdef"

func load(t *testing.T, env map[string]string) (Config, error) {
	t.Helper()
	for _, name := range []string{"ADDR", "DATA", "TLS_CERT", "TLS_KEY", "ACME_DOMAINS", "DESKTOP_KEY"} {
		t.Setenv(EnvPrefix+name, "")
	}
	t.Setenv(EnvPrefix+"DATA", t.TempDir())
	for k, v := range env {
		t.Setenv(EnvPrefix+k, v)
	}
	return Load()
}

func TestDesktopKeyIsOnlyAcceptedOnALoopbackAddress(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:8750", "[::1]:8750", "127.0.0.1:0"} {
		c, err := load(t, map[string]string{"ADDR": addr, "DESKTOP_KEY": goodKey})
		if err != nil {
			t.Errorf("%s: %v", addr, err)
			continue
		}
		if !c.Desktop() {
			t.Errorf("%s: desktop mode is off with a key set", addr)
		}
	}
	for _, addr := range []string{"0.0.0.0:8750", ":8750", "192.168.1.20:8750", "localhost:8750", "example.com:8750", "[::]:8750"} {
		if _, err := load(t, map[string]string{"ADDR": addr, "DESKTOP_KEY": goodKey}); err == nil || !strings.Contains(err.Error(), "loopback") {
			t.Errorf("%s: a desktop key was accepted on a non-loopback address (err %v)", addr, err)
		}
	}
}

func TestDesktopKeyNeedsLengthAndNoTLS(t *testing.T) {
	if _, err := load(t, map[string]string{"ADDR": "127.0.0.1:8750", "DESKTOP_KEY": goodKey[:31]}); err == nil {
		t.Error("a 31 character desktop key was accepted")
	}
	if _, err := load(t, map[string]string{"ADDR": "127.0.0.1:8750", "DESKTOP_KEY": goodKey, "ACME_DOMAINS": "example.com"}); err == nil {
		t.Error("a desktop key was accepted together with automatic HTTPS")
	}
	if _, err := load(t, map[string]string{"ADDR": "127.0.0.1:8750", "DESKTOP_KEY": goodKey, "TLS_CERT": "c.pem", "TLS_KEY": "k.pem"}); err == nil {
		t.Error("a desktop key was accepted together with TLS")
	}
}

func TestWithoutAKeyTheServerIsNotInDesktopMode(t *testing.T) {
	c, err := load(t, map[string]string{"ADDR": "0.0.0.0:8750"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Desktop() || c.DesktopKey != "" {
		t.Fatal("desktop mode is on without a key")
	}
}
