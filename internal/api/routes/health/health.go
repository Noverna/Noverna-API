package health

import (
	"net/http"
	"time"

	"noverna.de/m/v2/internal/api"
)

func Register(s *api.Server) {
	s.GetRouter().Get("/health", healthHandler(s))
}

func healthHandler(s *api.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}
	s.WriteJSON(w, http.StatusOK, response)
	}
}