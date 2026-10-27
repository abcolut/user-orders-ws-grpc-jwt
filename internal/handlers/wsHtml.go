package handlers

import (
	"log"
	"my_api/internal/api"
	"net/http"
	"os"
)

func WsHtmlHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ws_html" {
		api.WriteResponse(w, http.StatusNotFound, api.APIResponse{
			Error: &api.APIError{Code: "NOT_FOUND", Message: "Путь не ws"},
		})
		return
	}

	data, err := os.ReadFile("templates/ws.html")
	//data, err := os.ReadFile("templates/ws_old2.html")
	if err != nil {
		log.Printf("Ошибка чтения файла: %v", err)
		http.Error(w, "Не удалось прочитать шаблон", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
