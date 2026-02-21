package middleware

import (
	"net/http"
)

const (
	// MaxRequestBodySize is the maximum allowed request body size (1 MB).
	MaxRequestBodySize = 1 << 20
)

// MaxBodySize limits the size of incoming request bodies. Requests that
// exceed the limit will receive an HTTP 413 error when the handler
// attempts to read beyond the allowed bytes.
func MaxBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
		next.ServeHTTP(w, r)
	})
}
