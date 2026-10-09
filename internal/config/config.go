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

	KeyFile string

	ACMEDomains   []string
	ACMEEmail     string
	ACMEDirectory string
	HTTPAddr      string

	FetchPrivate bool
}

type Var struct {
	Name    string
	Default string
	Purpose string
}

var Vars = []Var{
	{"ADDR", "127.0.0.1:8750", "listen address; set 0.0.0.0:8750 to accept connections from other machines"},
	{"DATA", "./data", "data directory holding the database and uploads"},
	{"TLS_CERT", "", "path to a TLS certificate; set together with FORMMD_TLS_KEY"},
	{"TLS_KEY", "", "path to the TLS private key; set together with FORMMD_TLS_CERT"},
	{"KEY_FILE", "", "path to the at-rest encryption key (form-md keygen makes one); keep it outside the data directory. A systemd credential named form-md.key is used when this is unset"},
	{"ACME_DOMAINS", "", "comma-separated host names to get certificates for automatically (Let's Encrypt by default); needs ports 443 and 80 reachable"},
	{"ACME_EMAIL", "", "contact address given to the certificate authority"},
	{"ACME_DIRECTORY", "https://acme-v02.api.letsencrypt.org/directory", "ACME directory URL; point it at a staging or private CA to test"},
	{"HTTP_ADDR", ":80", "with ACME on, the plain HTTP listener that answers challenges and redirects to HTTPS"},
	{"FETCH_PRIVATE", "", "set to 1 to let link previews and media imports reach private network addresses such as intranet sites; off by default"},
}

func Load() (Config, error) {
	c := Config{
		Addr:    get("ADDR"),
		DataDir: get("DATA"),
		TLSCert: get("TLS_CERT"),
		TLSKey:  get("TLS_KEY"),

		KeyFile: get("KEY_FILE"),

		ACMEEmail:     get("ACME_EMAIL"),
		ACMEDirectory: get("ACME_DIRECTORY"),
		HTTPAddr:      get("HTTP_ADDR"),

		FetchPrivate: get("FETCH_PRIVATE") == "1" || get("FETCH_PRIVATE") == "true",
	}
	for _, d := range strings.Split(get("ACME_DOMAINS"), ",") {
		if d = strings.ToLower(strings.TrimSpace(d)); d != "" {
			c.ACMEDomains = append(c.ACMEDomains, d)
		}
	}
	if len(c.ACMEDomains) > 0 && c.TLSCert != "" {
		return c, errors.New(EnvPrefix + "ACME_DOMAINS and " + EnvPrefix + "TLS_CERT cannot both be set")
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		return c, fmt.Errorf("%sADDR %q: %w", EnvPrefix, c.Addr, err)
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
