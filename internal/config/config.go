package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const EnvPrefix = "FORMMD_"

type Config struct {
	Addr    string
	DataDir string
	TLSCert string
	TLSKey  string

	KeyFile   string
	UpdateURL string

	ACMEDomains   []string
	ACMEEmail     string
	ACMEDirectory string
	HTTPAddr      string

	FetchPrivate bool

	Family string

	DesktopKey string
}

type Var struct {
	Name    string
	Default string
	Purpose string
}

var Vars = []Var{
	{"ADDR", "127.0.0.1:8750", "listen address; set 0.0.0.0:8750 to accept connections from other machines"},
	{"DATA", "./data", "data directory holding the database and uploads"},
	{"TERMINAL", "", "on or off: the in-app terminal and Claude Code for the owner; unset means on only when ADDR is a loopback address"},
	{"CLAUDE_BIN", "", "path to the claude executable when it is not on the PATH or in the usual install folders"},
	{"AGENT_DIR", "", "folder Claude Code starts in; unset means DATA/agent"},
	{"MCP_AUTOSTART", "", "0 stops form-md mcp from starting a local server that is not running"},
	{"TLS_CERT", "", "path to a TLS certificate; set together with FORMMD_TLS_KEY"},
	{"TLS_KEY", "", "path to the TLS private key; set together with FORMMD_TLS_CERT"},
	{"KEY_FILE", "", "path to the at-rest encryption key (form-md keygen makes one); keep it outside the data directory. A systemd credential named form-md.key is used when this is unset"},
	{"ACME_DOMAINS", "", "comma-separated host names to get certificates for automatically (Let's Encrypt by default); needs ports 443 and 80 reachable"},
	{"ACME_EMAIL", "", "contact address given to the certificate authority"},
	{"ACME_DIRECTORY", "https://acme-v02.api.letsencrypt.org/directory", "ACME directory URL; point it at a staging or private CA to test"},
	{"HTTP_ADDR", ":80", "with ACME on, the plain HTTP listener that answers challenges and redirects to HTTPS"},
	{"CLOUDFLARED", "", "path to cloudflared for the public tunnel (looked up on PATH when unset)"},
	{"UPDATE_URL", "", "where the daily update check reads the latest release (GitHub's API when unset); point it at a mirror on a closed network"},
	{"FAMILY", "", "where the other form family apps run on this computer, as product=url pairs separated by commas (formlit=http://127.0.0.1:8390,formvis=http://127.0.0.1:8190 are assumed); loopback addresses only; off stops the lookups"},
	{"FETCH_PRIVATE", "", "set to 1 to let link previews and media imports reach private network addresses such as intranet sites; off by default"},
	{"DESKTOP_KEY", "", "set by the desktop app for the server it starts: the key its own window signs the owner in with; keeps the server on this computer. Leave unset"},
}

func Load() (Config, error) {
	c := Config{
		Addr:    get("ADDR"),
		DataDir: get("DATA"),
		TLSCert: get("TLS_CERT"),
		TLSKey:  get("TLS_KEY"),

		KeyFile:   get("KEY_FILE"),
		UpdateURL: get("UPDATE_URL"),

		ACMEEmail:     get("ACME_EMAIL"),
		ACMEDirectory: get("ACME_DIRECTORY"),
		HTTPAddr:      get("HTTP_ADDR"),

		FetchPrivate: get("FETCH_PRIVATE") == "1" || get("FETCH_PRIVATE") == "true",

		Family: get("FAMILY"),

		DesktopKey: get("DESKTOP_KEY"),
	}
	for _, d := range strings.Split(get("ACME_DOMAINS"), ",") {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" {
			c.ACMEDomains = append(c.ACMEDomains, d)
		}
	}
	if len(c.ACMEDomains) > 0 && c.TLSCert != "" {
		return c, errors.New(EnvPrefix + "ACME_DOMAINS and " + EnvPrefix + "TLS_CERT cannot both be set")
	}
	host, _, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return c, fmt.Errorf("%sADDR %q: %w", EnvPrefix, c.Addr, err)
	}
	if c.DesktopKey != "" {
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() || c.TLS() {
			return c, errors.New(EnvPrefix + "DESKTOP_KEY needs a loopback " + EnvPrefix + "ADDR without TLS")
		}
		if len(c.DesktopKey) < 32 {
			return c, errors.New(EnvPrefix + "DESKTOP_KEY must be at least 32 characters")
		}
	}
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return c, errors.New(EnvPrefix + "TLS_CERT and " + EnvPrefix + "TLS_KEY must be set together")
	}
	abs, err := filepath.Abs(c.DataDir)
	if err != nil {
		return c, fmt.Errorf("%sDATA %q: %w", EnvPrefix, c.DataDir, err)
	}
	c.DataDir = abs
	return c, nil
}

func (c Config) TLS() bool {
	return c.TLSCert != "" || len(c.ACMEDomains) > 0
}

func (c Config) ACME() bool {
	return len(c.ACMEDomains) > 0
}

func (c Config) Desktop() bool {
	return c.DesktopKey != ""
}

func get(name string) string {
	if v, ok := os.LookupEnv(EnvPrefix + name); ok && v != "" {
		return v
	}
	for _, v := range Vars {
		if v.Name == name {
			return v.Default
		}
	}
	return ""
}
