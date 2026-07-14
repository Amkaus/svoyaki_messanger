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

type IncomingEvent struct {
 Type        string `json:"type"`          
 ChatID      int    `json:"chat_id"`       
 Text        string `json:"text"`          
 ClientMsgID string `json:"client_msg_id"`
 MessageID   int    `json:"message_id"`
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
   var event IncomingEvent
   if err := conn.ReadJSON(&event); err != nil {
    break
   }

   switch event.Type {
   case "send":
    savedMsg, err := SaveMessage(db, event.ChatID, userID, event.Text, event.ClientMsgID)

    if err != nil {
     log.Println("Ошибка сохранения:", err)
     continue
    }
    broadcastEvent := map[string]interface{}{
     "type":    "message",
     "message": savedMsg,
    }
    broadcast(broadcastEvent)
   case "read":
    db.Exec(`INSERT INTO message_reads (message_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, event.MessageID, userID)
    broadcastEvent := map[string]interface{}{
     "type":       "read_receipt",
     "message_id": event.MessageID,
     "user_id":    userID,
    }
    broadcast(broadcastEvent)
   }
  }
 }
}

func broadcast(data interface{}) {
 clientsMu.Lock()
 defer clientsMu.Unlock()
 for clientConn := range clients {
  clientConn.WriteJSON(data)
 }
}
