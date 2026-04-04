package pages

import (
	"log"
	"net/http"
	"time"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		log.Printf("➡️  [REQUEST] %s %s", r.Method, r.URL.Path)
		
		// Passe la requête à la route finale (ou au prochain middleware)
		next.ServeHTTP(w, r)
		
		log.Printf("✅ [SERVED] %s %s (%v)", r.Method, r.URL.Path, time.Since(start))
	})
}
