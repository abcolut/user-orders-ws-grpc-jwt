package auth

import (
	// "encoding/json"
	"log"
	// "net"
	"net/http"
	// "os"

	"my_api/internal/api"
	// myDb "my_api/internal/db"
	// "my_api/internal/handlers"
	// "my_api/internal/ws"
	// pb "my_api/proto"

	"context"
	"github.com/golang-jwt/jwt/v5"
	// "github.com/joho/godotenv"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
	"strconv"
	"strings"
)

func AuthMiddleware(secret []byte, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			api.WriteResponse(w, http.StatusUnauthorized, api.APIResponse{
				Error: &api.APIError{Code: "JWT_ERROR", Message: "Missing Authorization header"},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			api.WriteResponse(w, http.StatusUnauthorized, api.APIResponse{
				Error: &api.APIError{Code: "JWT_ERROR", Message: "Invalid Authorization format"},
			})
			return
		}
		tokenString := parts[1]

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})
		if err != nil || !token.Valid {
			log.Printf("[Auth] Invalid token: %v", err)
			api.WriteResponse(w, http.StatusUnauthorized, api.APIResponse{
				Error: &api.APIError{Code: "JWT_INVALID", Message: "Invalid or expired token"},
			})
			return
		}

		var userID int
		found := false

		for _, key := range []string{"sub", "user_id", "id"} {
			if val, ok := claims[key]; ok {
				switch v := val.(type) {
				case float64:
					userID = int(v)
					found = true
					// log.Printf("[Auth] ✅ Найдено в ключе '%s': float64 -> int %d", key, userID)
				case string:
					idInt, e := strconv.Atoi(v)
					if e == nil {
						userID = idInt
						found = true
						// log.Printf("[Auth] ✅ Найдено в ключе '%s': string -> int %d", key, userID)
					}
				}
				if found {
					break
				}
			}
		}

		if !found {
			// log.Printf("[Auth] ❌ Не удалось найти userID ни в одном из ключей (sub, user_id, id)")
			api.WriteResponse(w, http.StatusUnauthorized, api.APIResponse{
				Error: &api.APIError{Code: "JWT_ERROR", Message: "Missing or invalid user ID in token"},
			})
			return
		}

		ctx := context.WithValue(r.Context(), "userID", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
