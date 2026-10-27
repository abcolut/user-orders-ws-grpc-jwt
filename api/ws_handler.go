// api/ws_handler.go
package api

// import (
// 	"log"
// 	"net/http"

// 	"github.com/gorilla/websocket"
// 	"my_api/internal/ws"
// )

// var upgrader = websocket.Upgrader{
// 	ReadBufferSize:  1024,
// 	WriteBufferSize: 1024,
// 	CheckOrigin: func(r *http.Request) bool {
// 		// Для учебного проекта разрешаем все источники.
// 		// В продакшене тут должна быть проверка Origin.
// 		return true
// 	},
// }

// func HandleWebSocket(room *ws.Room) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		conn, err := upgrader.Upgrade(w, r, nil)
// 		if err != nil {
// 			log.Printf("WebSocket upgrade error: %v", err)
// 			http.Error(w, "Failed to upgrade to WebSocket", http.StatusBadRequest)
// 			return
// 		}

// 		room.AddClient(conn)
// 		defer room.RemoveClient(conn)

// 		// Можно здесь же запустить горутину, которая слушает сообщения от клиента,
// 		// если клиенту нужно что-то слать на сервер. Для «push-уведомлений» это не обязательно.

// 		// Держим соединение открытым, пока клиент не отключится
// 		for {
// 			_, _, err := conn.ReadMessage()
// 			if err != nil {
// 				break // Клиент отключился
// 			}
// 		}
// 	}
// }
