package handlers

import (
	"log"
	"my_api/internal/api"
	"net/http"
	"os"
)

func AuthHtmlHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/auth" {
		api.WriteResponse(w, http.StatusNotFound, api.APIResponse{
			Error: &api.APIError{Code: "NOT_FOUND", Message: "Путь не auth"},
		})
		return
	}

	data, err := os.ReadFile("templates/auth.html")
	if err != nil {
		log.Printf("Ошибка чтения файла: %v", err)
		http.Error(w, "Не удалось прочитать шаблон", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	dataHeader, _ := os.ReadFile("templates/_header.html")
	w.Write(dataHeader)

	w.Write(data)

	dataFooter, _ := os.ReadFile("templates/_footer.html")
	w.Write(dataFooter)
}
