package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
    "log"
)

type AddMemberReq struct {
	ChatID     int `json:"chat_id"`
	Username string `json:"username"`
}

func AddMember(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        adminID := int(r.Context().Value("user_id").(float64))
        var req AddMemberReq
        
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
            return
        }

        var isAdmin bool
        err := db.QueryRow(`SELECT is_admin FROM chat_members WHERE chat_id = $1 AND user_id = $2`, req.ChatID, adminID).Scan(&isAdmin)
        if err != nil || !isAdmin {
            http.Error(w, "У вас нет прав администратора в этом чате", http.StatusForbidden)
            return
        }

        var newUserID int
        err = db.QueryRow(`SELECT id FROM users WHERE username = $1`, req.Username).Scan(&newUserID)
        if err != nil {
            if err == sql.ErrNoRows {
                http.Error(w, "Пользователь с таким логином не найден", http.StatusNotFound)
            } else {
                http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
            }
            return
        }

        _, err = db.Exec(`INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)`, req.ChatID, newUserID)
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
            SELECT m.id, m.sender_id, u.username, m.content, m.created_at,
                   EXISTS(SELECT 1 FROM message_reads mr WHERE mr.message_id = m.id AND mr.user_id != m.sender_id) AS is_read
            FROM messages m 
            JOIN users u ON m.sender_id = u.id
            WHERE m.chat_id = $1 AND m.id < $2 
            ORDER BY m.id DESC LIMIT 50
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
            err := rows.Scan(&msg.ID, &msg.SenderID, &msg.SenderName, &msg.Content, &msg.CreatedAt, &msg.IsRead)
            if err != nil {
                log.Println("Ошибка при сканировании строки:", err)
                continue
            }
            messages = append(messages, msg)
        }

        if messages == nil {
            messages = []MessageResponse{}
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(messages)
    }
}

func SaveMessage(db *sql.DB, chatID, senderID int, content string, clientMsgID string) (MessageResponse, error) {
	var msg MessageResponse
	
	query := `
        INSERT INTO messages (chat_id, sender_id, content, client_msg_id) 
        VALUES ($1, $2, $3, $4) 
        ON CONFLICT (client_msg_id) DO UPDATE SET chat_id = EXCLUDED.chat_id
        RETURNING id, chat_id, sender_id, (SELECT username FROM users WHERE id = $2), content, created_at, client_msg_id
    `
	err := db.QueryRow(query, chatID, senderID, content, clientMsgID).Scan(
		&msg.ID, &msg.ChatID, &msg.SenderID, &msg.SenderName, &msg.Content, &msg.CreatedAt, &msg.ClientMsgID,
	)
	
	if err != nil {
		return msg, err
	}
	
	return msg, nil
}

type RemoveMemberReq struct {
	ChatID int `json:"chat_id"`
	Username string `json:"username"`
}

func RemoveMember(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminID := int(r.Context().Value("user_id").(float64))
		var req RemoveMemberReq

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
			return
		}

		var isAdmin bool
		err := db.QueryRow(`SELECT is_admin FROM chat_members WHERE chat_id = $1 AND user_id = $2`, req.ChatID, adminID).Scan(&isAdmin)
		
		if err != nil || !isAdmin {
			http.Error(w, "У вас нет прав администратора для удаления участников", http.StatusForbidden)
			return
		}
		var targetUserID int
        err = db.QueryRow(`SELECT id FROM users WHERE username = $1`, req.Username).Scan(&targetUserID)
        if err != nil {
            if err == sql.ErrNoRows {
                http.Error(w, "Пользователь с таким логином не найден", http.StatusNotFound)
            } else {
                http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
            }
            return
        }

        if adminID == targetUserID {
            http.Error(w, "Нельзя удалить самого себя", http.StatusBadRequest)
            return
        }

		result, err := db.Exec(`DELETE FROM chat_members WHERE chat_id = $1 AND user_id = $2`, req.ChatID, targetUserID)
		if err != nil {
			http.Error(w, "Ошибка при удалении участника", http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, "Пользователь не найден в этом чате", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Пользователь успешно удален из чата"}`))
	}
}
