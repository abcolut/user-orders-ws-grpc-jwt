package handlers

import (
	"my_api/internal/api"
	"net/http"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/hello" {
		api.WriteResponse(w, http.StatusNotFound, api.APIResponse{
			Error: &api.APIError{Code: "NOT_FOUND", Message: "Путь не hello"},
		})
		return
	}

	api.WriteResponse(w, http.StatusOK, api.APIResponse{
		Data: map[string]string{
			"message": "Pet server - ws - db",
		},
	})
}
