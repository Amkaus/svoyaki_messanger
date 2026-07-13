package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type AddMemberReq struct {
	ChatID     int `json:"chat_id"`
	NewUserID  int `json:"new_user_id"`
}

func AddMember(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminID := int(r.Context().Value("user_id").(float64))
		var req AddMemberReq
		json.NewDecoder(r.Body).Decode(&req)
		var isAdmin bool

		err := db.QueryRow(`SELECT is_admin FROM chat_members WHERE chat_id = $1 AND user_id = $2`, req.ChatID, adminID).Scan(&isAdmin)
		if err != nil || !isAdmin {
			http.Error(w, "У вас нет прав администратора в этом чате", http.StatusForbidden)
			return
		}

		_, err = db.Exec(`INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)`, req.ChatID, req.NewUserID)
		if err != nil {
			http.Error(w, "Ошибка добавления (возможно, он уже в чате)", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"message": "Пользователь успешно добавлен"}`))
	}
}

func GetHistory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		chatID := r.URL.Query().Get("chat_id")
		lastID := r.URL.Query().Get("last_id") 

		if lastID == "" {
			lastID = "999999999"
		}
		
		query := `
			SELECT id, sender_id, content, created_at 
			FROM messages 
			WHERE chat_id = $1 AND id < $2 
			ORDER BY id DESC LIMIT 50
		`
		rows, err := db.Query(query, chatID, lastID)
		if err != nil {
			http.Error(w, "Ошибка получения истории", http.StatusInternalServerError)
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
