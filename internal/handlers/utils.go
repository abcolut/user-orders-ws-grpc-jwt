package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

type errorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func WriteJSONError(w http.ResponseWriter, msg string, status int) {
	WriteJSONErrorWithDetails(w, msg, "", status)
}

func WriteJSONErrorWithDetails(w http.ResponseWriter, msg, details string, status int) {
	resp := errorResponse{
		Error:   msg,
		Details: details,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Не удалось записать JSON-ответ об ошибке: %v", err)
	}
}
