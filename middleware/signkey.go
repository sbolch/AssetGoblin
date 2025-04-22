package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

type Signkey struct {
	Secret string
}

func (s *Signkey) Verify(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if !s.isValidToken(req.URL.Path, req.URL.Query().Get("token")) {
			http.Error(res, "Unauthorized", http.StatusUnauthorized)
			return
		}

		handler.ServeHTTP(res, req)
	})
}

func (s *Signkey) isValidToken(path, token string) bool {
	hasher := hmac.New(sha256.New, []byte(s.Secret))
	hasher.Write([]byte(path))
	expectedToken := hex.EncodeToString(hasher.Sum(nil))
	return hmac.Equal([]byte(token), []byte(expectedToken))
}
