package httpx

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var gzPool = sync.Pool{New: func() any {
	zw, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
	return zw
}}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		name, _, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(name, "gzip") {
			return true
		}
	}
	return false
}

func JSONZ(w http.ResponseWriter, r *http.Request, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		Error(w, http.StatusInternalServerError, "internal", "could not encode the response")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Add("Vary", "Accept-Encoding")
	if buf.Len() < 1400 || !acceptsGzip(r) {
		h.Set("Content-Length", strconv.Itoa(buf.Len()))
		w.WriteHeader(status)
		_, _ = w.Write(buf.Bytes())
		return
	}
	zw := gzPool.Get().(*gzip.Writer)
	defer gzPool.Put(zw)
	var out bytes.Buffer
	zw.Reset(&out)
	_, _ = zw.Write(buf.Bytes())
	_ = zw.Close()
	h.Set("Content-Encoding", "gzip")
	h.Set("Content-Length", strconv.Itoa(out.Len()))
	w.WriteHeader(status)
	_, _ = w.Write(out.Bytes())
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}
