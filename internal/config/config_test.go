package config

import (
	"strings"
	"testing"
)

const goodKey = "0123456789abcdef0123456789abcdef"

func load(t *testing.T, env map[string]string) (Config, error) {
	t.Helper()
	for _, name := range []string{"ADDR", "DATA", "TLS_CERT", "TLS_KEY", "ACME_DOMAINS", "DESKTOP", "DESKTOP_KEY"} {
		t.Setenv(EnvPrefix+name, "")
	}
	t.Setenv(EnvPrefix+"DATA", t.TempDir())
	for k, v := range env {
		t.Setenv(EnvPrefix+k, v)
	}
	return Load()
}

func TestDesktopModeIsOnlyAcceptedOnALoopbackAddress(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:8750", "[::1]:8750", "127.0.0.1:0"} {
		c, err := load(t, map[string]string{"ADDR": addr, "DESKTOP": "1"})
		if err != nil {
			t.Errorf("%s: %v", addr, err)
			continue
		}
		if !c.Desktop() {
			t.Errorf("%s: desktop mode is off with FORMMD_DESKTOP=1", addr)
		}
	}
	for _, addr := range []string{"0.0.0.0:8750", ":8750", "192.168.1.20:8750", "localhost:8750", "example.com:8750", "[::]:8750"} {
		if _, err := load(t, map[string]string{"ADDR": addr, "DESKTOP": "1"}); err == nil || !strings.Contains(err.Error(), "loopback") {
			t.Errorf("%s: desktop mode was accepted on a non-loopback address (err %v)", addr, err)
		}
	}
}

func TestDesktopModeNeedsNoTLS(t *testing.T) {
	if _, err := load(t, map[string]string{"ADDR": "127.0.0.1:8750", "DESKTOP": "1", "ACME_DOMAINS": "example.com"}); err == nil {
		t.Error("desktop mode was accepted together with automatic HTTPS")
	}
	if _, err := load(t, map[string]string{"ADDR": "127.0.0.1:8750", "DESKTOP": "1", "TLS_CERT": "c.pem", "TLS_KEY": "k.pem"}); err == nil {
		t.Error("desktop mode was accepted together with TLS")
	}
}

func TestTheServerMakesTheWindowKeyAndTakesNoneFromOutside(t *testing.T) {
	a, err := load(t, map[string]string{"ADDR": "127.0.0.1:8750", "DESKTOP": "1", "DESKTOP_KEY": goodKey})
	if err != nil {
		t.Fatal(err)
	}
	b, err := load(t, map[string]string{"ADDR": "127.0.0.1:8750", "DESKTOP": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if a.DesktopKey == goodKey || len(a.DesktopKey) < 32 || len(b.DesktopKey) < 32 || a.DesktopKey == b.DesktopKey {
		t.Fatalf("window keys %q and %q: want a new key of at least 32 characters at every start, never the one in the environment", a.DesktopKey, b.DesktopKey)
	}
	if c, err := load(t, map[string]string{"ADDR": "127.0.0.1:8750", "DESKTOP_KEY": goodKey}); err != nil || c.Desktop() {
		t.Fatalf("FORMMD_DESKTOP_KEY alone turned desktop mode on (err %v)", err)
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
