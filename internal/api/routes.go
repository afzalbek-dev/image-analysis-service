package api

import (
	"log"
	"net/http"
	"time"
)

// SetupRouter registers application routes and wraps them in middleware.
func (s *Server) SetupRouter() http.Handler {
	mux := http.NewServeMux()

	// System & Health
	mux.HandleFunc("/health", s.HandleHealth)
	mux.HandleFunc("/healthz", s.HandleHealth)

	// API v1
	mux.HandleFunc("/api/v1/analyze", s.HandleAnalyze)
	mux.HandleFunc("/api/v1/duplicates/register", s.HandleDuplicateRegister)
	mux.HandleFunc("/api/v1/duplicates/check", s.HandleDuplicateCheck)
	mux.HandleFunc("/api/v1/process/transform", s.HandleTransform)

	return s.loggingAndRecoveryMiddleware(mux)
}

func (s *Server) loggingAndRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC RECOVERED] %s %s: %v", r.Method, r.URL.Path, rec)
				s.writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()

		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s - %v", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}
