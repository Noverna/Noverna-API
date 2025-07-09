package routes

import (
	"noverna.de/m/v2/internal/api"
	"noverna.de/m/v2/internal/api/routes/health"
)

func SetupRoutes(s *api.Server) {
	/// Setup all Routes
	health.Register(s)
}