package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignkey_Verify(t *testing.T) {
	tests := []struct {
		name       string
		secret     string
		path       string
		token      string
		wantStatus int
	}{
		{
			name:       "valid token",
			secret:     "secret",
			path:       "/test-path",
			token:      generateToken("secret", "/test-path"),
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid token",
			secret:     "secret",
			path:       "/test-path",
			token:      "invalid-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty token",
			secret:     "secret",
			path:       "/test-path",
			token:      "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong secret",
			secret:     "wrong-secret",
			path:       "/test-path",
			token:      generateToken("secret", "/test-path"),
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signkey := Signkey{Secret: tt.secret}
			handler := signkey.Verify(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path+"?token="+tt.token, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("Verify() = status %v, want %v", status, tt.wantStatus)
			}
		})
	}
}

func generateToken(secret, path string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(path))
	return hex.EncodeToString(mac.Sum(nil))
}
