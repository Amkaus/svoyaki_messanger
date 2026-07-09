package middleware

import (
 "context"
 "net/http"
 "strings"
 "github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("super_secret_key_for_messenger")

func AuthMiddleware(n http.HandlerFunc) http.HandlerFunc {
 return func(w http.ResponseWriter, r *http.Request) {
  authHeader := r.Header.Get("Authorization")

  if authHeader == "" {
   http.Error(w, "Отсутствует токен авторизации", http.StatusUnauthorized)
   return
  }
  parts := strings.Split(authHeader, " ")

  if len(parts) != 2 || parts[0] != "Bearer" {
   http.Error(w, "Неверный формат заголовка", http.StatusUnauthorized)
   return
  }

  tokenString := parts[1]
  token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
   return jwtSecret, nil
  })

  if err != nil || !token.Valid {
   http.Error(w, "Недействительный токен", http.StatusUnauthorized)
   return
  }
  claims, ok := token.Claims.(jwt.MapClaims)

  if !ok {
   http.Error(w, "Ошибка чтения токена", http.StatusUnauthorized)
   return
  }

  ctx := context.WithValue(r.Context(), "user_id", claims["user_id"])
  n.ServeHTTP(w, r.WithContext(ctx))
 }
}


