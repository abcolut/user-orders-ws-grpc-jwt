package handlers

import (
	"my_api/internal/api"
	"net/http"
)

func SecretHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/secret" {
		api.WriteResponse(w, http.StatusNotFound, api.APIResponse{
			Error: &api.APIError{Code: "NOT_FOUND", Message: "Путь не secret"},
		})
		return
	}

	api.WriteResponse(w, http.StatusOK, api.APIResponse{
		Data: map[string]string{
			"message": "Pet server - secret page",
		},
	})
}
