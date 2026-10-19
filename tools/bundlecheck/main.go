package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var assetRef = regexp.MustCompile(`(?:src|href)="(/assets/[A-Za-z0-9._-]+\.js)"`)

func main() {
	dist := flag.String("dist", "web/dist", "the built client")
	budget := flag.Float64("budget", 262, "KB of gzipped JavaScript the first page may load before it runs")
	flag.Parse()
	html, err := os.ReadFile(filepath.Join(*dist, "index.html"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "bundlecheck:", err)
		os.Exit(1)
	}
	seen := map[string]bool{}
	total := 0
	for _, m := range assetRef.FindAllSubmatch(html, -1) {
		p := string(m[1])
		if seen[p] {
			continue
		}
		seen[p] = true
		raw, err := os.ReadFile(filepath.Join(*dist, strings.TrimPrefix(p, "/")))
		if err != nil {
			fmt.Fprintln(os.Stderr, "bundlecheck:", err)
			os.Exit(1)
		}
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		_, _ = zw.Write(raw)
		_ = zw.Close()
		total += buf.Len()
	}
	kb := float64(total) / 1024
	if kb > *budget {
		fmt.Printf("bundlecheck: %.1f KB gzip of JavaScript up front in %d files, over the %.0f KB budget\n", kb, len(seen), *budget)
		os.Exit(1)
	}
	fmt.Printf("bundlecheck: %.1f KB gzip of JavaScript up front in %d files, within %.0f KB\n", kb, len(seen), *budget)
}
