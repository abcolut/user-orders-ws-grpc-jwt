package api

import (
	"encoding/json"
	"errors"
	"fmt"
	validator "github.com/go-playground/validator/v10"
	"io"
	"log"
	"net/http"

	dbpkg "my_api/internal/db"
)

type APIResponse struct {
	Data  any       `json:"data,omitempty"`
	Error *APIError `json:"error,omitempty"`
}

type APIError struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

func WriteResponse(w http.ResponseWriter, status int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("encode error: %v", err)
	}
}

func ReadBody(r *http.Request) ([]byte, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения тела запроса: %w", err)
	}
	return bodyBytes, nil
}

// var v = validator.New()

func ValidateOrder(req dbpkg.OrderRequest) ([]string, error) {
	// func ValidateOrder(v *validator.Validate, req models.OrderRequest) ([]string, error) {
	var v = validator.New()
	err := v.Struct(req)
	if err == nil {
		return nil, nil
	}

	var messages []string
	var ve validator.ValidationErrors

	if errors.As(err, &ve) {
		for _, e := range ve {
			messages = append(messages, fmt.Sprintf("%s: %s", e.Field(), e.Tag()))
		}
	} else {
		messages = append(messages, err.Error())
	}

	return messages, nil
}
