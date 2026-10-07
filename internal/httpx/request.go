package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type APIError struct {
	Status  int
	Code    string
	Message string
	Data    map[string]any
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

func Fail(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

func (e *APIError) With(key string, v any) *APIError {
	if e.Data == nil {
		e.Data = map[string]any{}
	}
	e.Data[key] = v
	return e
}

var (
	ErrNotFound     = Fail(http.StatusNotFound, "not_found", "not found")
	ErrUnauthorized = Fail(http.StatusUnauthorized, "unauthorized", "sign in required")
	ErrForbidden    = Fail(http.StatusForbidden, "forbidden", "not allowed")
	ErrReadOnly     = Fail(http.StatusForbidden, "read_only", "this credential is read-only")
	ErrSessionOnly  = Fail(http.StatusForbidden, "session_only", "this action needs a signed-in browser session")
)

const MaxJSONBody = 8 << 20

func Decode(r *http.Request, v any) error {
	body := http.MaxBytesReader(nil, r.Body, MaxJSONBody)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return Fail(http.StatusRequestEntityTooLarge, "too_large", "request body too large")
		}
		return Fail(http.StatusBadRequest, "bad_json", "invalid request body: "+err.Error())
	}
	if _, err := dec.Token(); err != io.EOF {
		return Fail(http.StatusBadRequest, "bad_json", "invalid request body: trailing data")
	}
	return nil
}

func Problem(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	var api *APIError
	if errors.As(err, &api) {
		JSON(w, api.Status, errorBody{Error: errorDetail{Code: api.Code, Message: api.Message, Data: api.Data}})
		return
	}
	log.Error("request failed", "rid", RequestIDFrom(r.Context()), "path", r.URL.Path, "err", err)
	Error(w, http.StatusInternalServerError, "internal", "internal error")
}

func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if f := r.Header.Get("X-Forwarded-For"); f != "" {
			first, _, _ := strings.Cut(f, ",")
			if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
				return ip.String()
			}
		}
		if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-Ip"))); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func RequestScheme(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if p := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))); p == "https" || p == "http" {
			return p
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func RequestOrigin(r *http.Request, trustProxy bool) string {
	host := r.Host
	if trustProxy {
		if h := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); h != "" {
			host = h
		}
	}
	return RequestScheme(r, trustProxy) + "://" + host
}

func IsUnsafe(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	return true
}
