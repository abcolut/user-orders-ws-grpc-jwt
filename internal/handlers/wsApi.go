package handlers

import (
	"log"
	"net/http"
	"strconv"

	"my_api/internal/ws"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// для разработки можно разрешить всё; в проде — валидировать origin
		return true
	},
}

var Hub *ws.Hub

func SetHub(h *ws.Hub) {
	Hub = h
}

func WsApiHandler(w http.ResponseWriter, r *http.Request) {
	param := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(param)
	if err != nil || userID <= 0 {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := &ws.Client{
		Conn:   conn,
		UserID: userID,
	}

	Hub.AddClient(userID, client)
	defer Hub.RemoveClient(userID, client)

	// простой цикл чтения входящих сообщений (если нужно)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("ws read error: %v", err)
			break
		}
		// Обработать команды клиента
		_ = message
	}
}
