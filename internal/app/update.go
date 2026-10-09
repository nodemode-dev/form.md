package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const releasesURL = "https://api.github.com/repos/nodemode-ai/form.md/releases/latest"

var latest atomic.Pointer[string]

func LatestRelease() string {
	if v := latest.Load(); v != nil && Newer(*v, Version) {
		return *v
	}
	return ""
}

func CheckUpdates(ctx context.Context, url string, enabled func() bool, log *slog.Logger) {
	if url == "" {
		url = releasesURL
	}
	client := &http.Client{Timeout: 15 * time.Second}
	wait := 5 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = 24 * time.Hour
		if !enabled() {
			continue
		}
		tag, err := fetchTag(ctx, client, url)
		if err != nil {
			log.Info("update check failed", "err", err)
			wait = 6 * time.Hour
			continue
		}
		latest.Store(&tag)
		if Newer(tag, Version) {
			log.Info("a newer version is available", "latest", tag, "running", Version)
		}
	}
}

func fetchTag(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", Product+"/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &statusError{resp.StatusCode}
	}
	var body struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return "", err
	}
	if _, ok := parseVersion(body.Tag); !ok {
		return "", &statusError{0}
	}
	return body.Tag, nil
}

type statusError struct{ code int }

func (e *statusError) Error() string {
	if e.code == 0 {
		return "the release has no version tag"
	}
	return "release lookup answered " + strconv.Itoa(e.code)
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func Newer(a, b string) bool {
	x, ok1 := parseVersion(a)
	y, ok2 := parseVersion(b)
	if !ok1 || !ok2 {
		return false
	}
	for i := range 3 {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}
