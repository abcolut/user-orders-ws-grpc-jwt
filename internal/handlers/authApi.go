package handlers

import (
	"my_api/internal/api"
	"net/http"
	"time"

	"database/sql"
	"encoding/json"

	"github.com/golang-jwt/jwt/v5"
	bcrypt "golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func AuthApiHandler(db *sql.DB, secret []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			api.WriteResponse(w, http.StatusMethodNotAllowed, api.APIResponse{
				Error: &api.APIError{Code: "METHOD_NOT_ALLOWED", Message: "Только POST"},
			})
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.WriteResponse(w, http.StatusBadRequest, api.APIResponse{
				Error: &api.APIError{Code: "INVALID_REQUEST", Message: "Некорректный JSON"},
			})
			return
		}

		var userID int
		var storedHash string
		err := db.QueryRowContext(r.Context(), `SELECT id, hashed_password FROM users WHERE username = $1`, req.Username).Scan(&userID, &storedHash)
		if err == sql.ErrNoRows {
			api.WriteResponse(w, http.StatusUnauthorized, api.APIResponse{
				Error: &api.APIError{Code: "AUTH_FAILED", Message: "Неверный логин или пароль"},
			})
			return
		}
		if err != nil {
			api.WriteResponse(w, http.StatusInternalServerError, api.APIResponse{
				Error: &api.APIError{Code: "DB_ERROR", Message: "Ошибка базы данных"},
			})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)); err != nil {
			api.WriteResponse(w, http.StatusUnauthorized, api.APIResponse{
				Error: &api.APIError{Code: "AUTH_FAILED", Message: "Неверный логин или пароль"},
			})
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  userID,
			"username": req.Username,
			"exp":      time.Now().Add(24 * time.Hour).Unix(), // срок жизни токена
		})

		tokenString, err := token.SignedString(secret)
		if err != nil {
			api.WriteResponse(w, http.StatusInternalServerError, api.APIResponse{
				Error: &api.APIError{Code: "JWT_ERROR", Message: "Ошибка создания токена"},
			})
			return
		}

		// 4. Возвращаем ответ
		api.WriteResponse(w, http.StatusOK, api.APIResponse{
			Data: map[string]any{
				"access_token": tokenString,
				"token_type":   "Bearer",
				"expires_in":   86400, // 24 часа в секундах
			},
		})
	}
}
