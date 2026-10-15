package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// JSON envelope responses.
type response struct {
	Code    int    `json:"code"` // 0 = success; otherwise HTTP-ish error code
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

const (
	codeSuccess = 0
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func ok(w http.ResponseWriter, data any, message string) {
	writeJSON(w, http.StatusOK, response{Code: codeSuccess, Message: message, Data: data})
}

func fail(w http.ResponseWriter, status, code int, message string) {
	writeJSON(w, status, response{Code: code, Message: message})
}

func decodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("empty request body")
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

// authToken extracts the bearer token from the Authorization header.
func authToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	h = strings.TrimSpace(h)
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return h
}

// pathInt parses an int path value (used for ?offset/&limit query params).
func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	n := 0
	sign := 1
	i := 0
	if s[0] == '-' {
		sign = -1
		i++
	}
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
	}
	return sign * n
}