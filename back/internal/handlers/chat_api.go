package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

type CreateChatReq struct {
	Name    string `json:"name"`
	IsGroup bool   `json:"is_group"`
}

func CreateChat(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := int(r.Context().Value("user_id").(float64)) 

		var req CreateChatReq
		json.NewDecoder(r.Body).Decode(&req)
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
			return
		}
		var chatID int
		err = tx.QueryRow(`INSERT INTO chats (name, is_group) VALUES ($1, $2) RETURNING id`, req.Name, req.IsGroup).Scan(&chatID)
		if err != nil {
			tx.Rollback() 
			http.Error(w, "Ошибка создания чата", http.StatusInternalServerError)
			return
		}
		_, err = tx.Exec(`INSERT INTO chat_members (chat_id, user_id, is_admin) VALUES ($1, $2, $3)`, chatID, userID, true)
		if err != nil {
			tx.Rollback() 
			http.Error(w, "Ошибка добавления создателя", http.StatusInternalServerError)
			return
		}
		tx.Commit()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"message": "Чат создан", "chat_id": %d}`, chatID)))
	}
}

type MessageResponse struct {
	ID        int    `json:"id"`
	SenderID  int    `json:"sender_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func Search(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chatID := r.URL.Query().Get("chat_id")
		searchText := r.URL.Query().Get("text")
		query := `
			SELECT id, sender_id, content, created_at 
			FROM messages 
			WHERE chat_id = $1 AND content ILIKE $2
			ORDER BY created_at DESC
		`		
		rows, err := db.Query(query, chatID, "%"+searchText+"%")
		if err != nil {
			http.Error(w, "Ошибка поиска", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var messages []MessageResponse
		for rows.Next() {
			var msg MessageResponse
			rows.Scan(&msg.ID, &msg.SenderID, &msg.Content, &msg.CreatedAt)
			messages = append(messages, msg)
		}

		if messages == nil {
			messages = []MessageResponse{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}
}