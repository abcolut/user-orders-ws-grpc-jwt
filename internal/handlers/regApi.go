package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"database/sql"
	"my_api/internal/api"
	dbpkg "my_api/internal/db"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func RegApiHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Только POST
		if r.Method != http.MethodPost {
			api.WriteResponse(w, http.StatusMethodNotAllowed, api.APIResponse{
				Error: &api.APIError{Code: "METHOD_NOT_ALLOWED", Message: "Разрешён только POST"},
			})
			return
		}

		var req dbpkg.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.WriteResponse(w, http.StatusBadRequest, api.APIResponse{
				Error: &api.APIError{Code: "INVALID_REQUEST", Message: "Неверный JSON"},
			})
			return
		}

		// Простая валидация
		if req.Username == "" || req.Password == "" {
			api.WriteResponse(w, http.StatusBadRequest, api.APIResponse{
				Error: &api.APIError{Code: "VALIDATION_ERROR", Message: "Username и password обязательны"},
			})
			return
		}

		id, err := dbpkg.CreateUser(db, req)

		if err != nil {
			log.Printf(err.Error())
			api.WriteResponse(w, http.StatusConflict, api.APIResponse{
				Error: &api.APIError{Code: "USER_EXISTS", Message: "Пользователь не создан"},
			})
			return
		}

		api.WriteResponse(w, http.StatusCreated, api.APIResponse{
			Data: map[string]string{
				"message":  "Пользователь успешно зарегистрирован ",
				"username": req.Username,
				"ID":       strconv.Itoa(id),
			},
		})
	}
}
