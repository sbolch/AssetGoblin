// Package middleware provides HTTP middleware components for the application.
// These middleware components can be used to add functionality like authentication,
// rate limiting, and other request processing to the HTTP server.
package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

// Signkey is a middleware that verifies request signatures using HMAC-SHA256.
// It requires a secret key to validate tokens provided in request query parameters.
type Signkey struct {
	Secret string // Secret is the key used for HMAC signature verification
}

// Verify returns a middleware handler that checks if the request has a valid
// signature token. If the token is valid, the request is passed to the next handler.
// If the token is invalid, a 401 Unauthorized response is returned.
func (s *Signkey) Verify(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if !s.isValidToken(req.URL.Path, req.URL.Query().Get("token")) {
			http.Error(res, "Unauthorized", http.StatusUnauthorized)
			return
		}

		handler.ServeHTTP(res, req)
	})
}

// isValidToken checks if the provided token is valid for the given path.
// It computes an HMAC-SHA256 hash of the path using the secret key and compares
// it with the provided token. Returns true if the token is valid, false otherwise.
func (s *Signkey) isValidToken(path, token string) bool {
	hasher := hmac.New(sha256.New, []byte(s.Secret))
	hasher.Write([]byte(path))
	expectedToken := hex.EncodeToString(hasher.Sum(nil))
	return hmac.Equal([]byte(token), []byte(expectedToken))
}
