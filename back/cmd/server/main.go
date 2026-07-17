package main

import (
 "database/sql"
 "fmt"
 "log"
 "net/http"

 "github.com/Amkaus/svoyaki_messanger/back/internal/handlers"
 "github.com/Amkaus/svoyaki_messanger/back/internal/middleware"
 _ "github.com/lib/pq"
)

func main() {
 connStr := "postgres://messenger_user:secretpassword@localhost:5433/messenger_db?sslmode=disable"
 db, err := sql.Open("postgres", connStr)

 if err != nil {
  log.Fatalf("Ошибка при подключении к БД: %v", err)
 }

 defer db.Close()

 if err := db.Ping(); err != nil {
  log.Fatalf("База данных недоступна: %v", err)
 }
 fmt.Println("Успешное подключение к PostgreSQL!")
 mux := http.NewServeMux()

 mux.HandleFunc("/register", handlers.Register(db))
 mux.HandleFunc("/login", handlers.Login(db))
 mux.HandleFunc("/chats", middleware.AuthMiddleware(handlers.GetChats(db)))
 mux.HandleFunc("/ws", middleware.AuthMiddleware(handlers.Subscribe(db)))
 mux.HandleFunc("/chats/create", middleware.AuthMiddleware(handlers.CreateChat(db)))
 mux.HandleFunc("/chats/add_member", middleware.AuthMiddleware(handlers.AddMember(db)))
 mux.HandleFunc("/chats/remove_member", middleware.AuthMiddleware(handlers.RemoveMember(db)))
 mux.HandleFunc("/messages/history", middleware.AuthMiddleware(handlers.GetHistory(db)))
 mux.HandleFunc("/messages/search", middleware.AuthMiddleware(handlers.Search(db)))
 mux.Handle("/", http.FileServer(http.Dir("./front")))
 mux.HandleFunc("/messages/read-all", middleware.AuthMiddleware(handlers.MarkChatAsRead(db)))

 protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  userID := r.Context().Value("user_id")
  w.Write([]byte(fmt.Sprintf("Привет, пользователь свояка №%v! C токеном все в порядке.", userID)))
 })

 mux.HandleFunc("/test-auth", middleware.AuthMiddleware(protectedHandler))
 fmt.Println("Сервер запущен на http://localhost:8080")

 if err := http.ListenAndServe(":8080", mux); err != nil {
  log.Fatalf("Ошибка запуска сервера: %v", err)
 }
}

