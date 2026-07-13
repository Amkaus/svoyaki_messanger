package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type ChatResponse struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	IsGroup bool   `json:"is_group"`
}

func GetChats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(float64)

		query := `
			SELECT c.id, COALESCE(c.name, 'Личный чат') as name, c.is_group 
			FROM chats c
			JOIN chat_members cm ON c.id = cm.chat_id
			WHERE cm.user_id = $1
		`
		
		rows, err := db.Query(query, int(userID))
		if err != nil {
			http.Error(w, "Ошибка при получении чатов", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var chats []ChatResponse
		for rows.Next() {
			var chat ChatResponse
			if err := rows.Scan(&chat.ID, &chat.Name, &chat.IsGroup); err != nil {
				continue
			}
			chats = append(chats, chat)
		}

		if chats == nil {
			chats = []ChatResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chats)
	}
}