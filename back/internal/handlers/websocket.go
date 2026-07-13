package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"sync"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool { return true },
}

type IncomingMessage struct {
	ChatID int    `json:"chat_id"`
	Text   string `json:"text"`
}

var (
	clients   = make(map[*websocket.Conn]int)
	clientsMu sync.Mutex
)

func Subscribe(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := int(r.Context().Value("user_id").(float64))
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			log.Println("Ошибка WS:", err)
			return
		}

		clientsMu.Lock()
		clients[conn] = userID
		clientsMu.Unlock()
		defer func() {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			conn.Close()
		}()

		for {
			var incMsg IncomingMessage
			err := conn.ReadJSON(&incMsg)

			if err != nil {
				break 
			}
			savedMsg, err := SaveMessage(db, incMsg.ChatID, userID, incMsg.Text)

			if err != nil {
				log.Println("Не удалось сохранить сообщение:", err)
				continue
			}
			clientsMu.Lock()

			for clientConn := range clients {
				err := clientConn.WriteJSON(savedMsg)
				
				if err != nil {
					clientConn.Close()
					delete(clients, clientConn)
				}
			}
			clientsMu.Unlock()
		}
	}
}

