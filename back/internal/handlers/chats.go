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
	IsAdmin bool   `json:"is_admin"`
}

func GetChats(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(float64)

		query := `
		SELECT c.id, 
			CASE 
				WHEN c.is_group THEN c.name 
				ELSE (
					SELECT u.username 
					FROM users u 
					JOIN chat_members cm2 ON u.id = cm2.user_id 
					WHERE cm2.chat_id = c.id AND u.id != $1 
					LIMIT 1
				)
			END as name,
			c.is_group, cm.is_admin
		FROM chats c
		JOIN chat_members cm ON c.id = cm.chat_id
		WHERE cm.user_id = $1`
		
		rows, err := db.Query(query, int(userID))
		if err != nil {
			http.Error(w, "Ошибка при получении чатов", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var chats []ChatResponse
		for rows.Next() {
			var chat ChatResponse
			if err := rows.Scan(&chat.ID, &chat.Name, &chat.IsGroup, &chat.IsAdmin); err != nil {
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