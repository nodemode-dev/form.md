package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

const EnvPrefix = "FORMMD_"

type Config struct {
	Addr    string
	DataDir string
	TLSCert string
	TLSKey  string
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
}

func Load() (Config, error) {
	c := Config{
		Addr:    get("ADDR"),
		DataDir: get("DATA"),
		TLSCert: get("TLS_CERT"),
		TLSKey:  get("TLS_KEY"),
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
	return c.TLSCert != ""
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
