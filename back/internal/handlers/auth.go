package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("super_secret_key_for_messenger")

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Register(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
			return
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Ошибка при обработке пароля", http.StatusInternalServerError)
			return
		}
		query := `INSERT INTO users (username, password_hash) VALUES ($1, $2)`
		_, err = db.Exec(query, req.Username, string(hashedPassword))
		if err != nil {
			http.Error(w, "Ошибка при создании пользователя", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"message": "Пользователь успешно зарегистрирован"}`))
	}
}

func Login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Неверный формат", http.StatusBadRequest)
			return
		}
		var id int
		var hash string
		err := db.QueryRow(`SELECT id, password_hash FROM users WHERE username = $1`, req.Username).Scan(&id, &hash)
		if err != nil {
			http.Error(w, "Пользователь не найден", http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
			http.Error(w, "Неверный пароль", http.StatusUnauthorized)
			return
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": id,
			"exp": time.Now().Add(24 * time.Hour).Unix(),
		})
		tokenString, _ := token.SignedString(jwtSecret)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
	}
}