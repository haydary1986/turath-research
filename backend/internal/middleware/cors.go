package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/cors"
)

func CORS(allowedOrigins string) func(next http.Handler) http.Handler {
	origins := []string{"*"}
	if allowedOrigins != "" && allowedOrigins != "*" {
		origins = strings.Split(allowedOrigins, ",")
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
