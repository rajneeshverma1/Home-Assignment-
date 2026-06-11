package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

// ErrorResponse defines standard JSON error format
type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	// Load environment variables if .env exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// Initialize subsystems
	InitDB()
	InitJWT()

	// Setup router
	r := chi.NewRouter()

	// Standard middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Configure CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept"},
		AllowCredentials: true,
		Debug:            false,
	})
	r.Use(c.Handler)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// Public Auth routes
	r.Post("/signup", SignupHandler)
	r.Post("/login", LoginHandler)

	// Protected task routes
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)

		r.Post("/tasks", CreateTaskHandler)
		r.Get("/tasks", GetTasksHandler)
		r.Get("/tasks/{id}", GetTaskHandler)
		r.Patch("/tasks/{id}", UpdateTaskHandler)
		r.Delete("/tasks/{id}", DeleteTaskHandler)
		r.Get("/tasks/{id}/logs", TaskLogsHandler)
	})

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// Global utility responders
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
