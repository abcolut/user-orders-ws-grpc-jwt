package auth

import (
	// "context"
	"net/http"
	"net/http/httptest"
	"testing"
	// "time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware(t *testing.T) {
	secret := []byte("test-secret-key")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := AuthMiddleware(secret, handler)

	t.Run("no-token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("ожидали 401, получили %d", rec.Code)
		}
	})

	t.Run("valid-token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "1", // user_id
		})
		tokenString, err := token.SignedString(secret)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()
		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("ожидали 200, получили %d", rec.Code)
		}
	})
}
