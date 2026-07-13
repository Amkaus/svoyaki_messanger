package handlers

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true 
	},
}

func Subscribe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(float64)
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Printf("Ошибка апгрейда до WebSocket: %v", err)
			return
		}
		defer conn.Close()

		fmt.Printf("Пользователь %.0f подключился к WebSocket!\n", userID)

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Пользователь %.0f отключился", userID)
				break
			}
			log.Printf("Получено сообщение от %.0f: %s", userID, string(message))
			response := []byte(fmt.Sprintf("Сервер получил твое сообщение: %s", string(message)))
			
			if err := conn.WriteMessage(messageType, response); err != nil {
				log.Println("Ошибка отправки ответа:", err)
				break
			}
		}
	}
}
