package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"noverna.de/m/v2/internal/config"
	"noverna.de/m/v2/internal/logger"
	custommw "noverna.de/m/v2/internal/middleware"
)

// Our API Server
type Server struct {
	config *config.Config
	router *chi.Mux
	httpServer *http.Server
	logger *logger.Logger
}

type APIResponse struct {
	Status int         `json:"status"`
	Data   any `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func NewServer(cfg *config.Config, log *logger.Logger) *Server {
	if cfg == nil {
		cfg = config.GetConfig()
	}

	if log == nil {
		log = logger.NewLogger()
		log.WithField("service", "API")
		log.WithField("component", "server")
		log.SetLevel(logger.INFO)
	}

	s := &Server{
		config: cfg,
		router: chi.NewRouter(),
		logger: log,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))

	s.router.Use(custommw.DetailedLoggerMiddleware(s.logger))

	// Simple Logging
	// s.router.Use(custommw.SimpleLoggerMiddleware(s.logger))

	s.router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
	}))
}

func (s *Server) setupRoutes() {
	s.router.Get("/", s.Index)
	s.router.Get("/health", s.Health)
	s.router.Get("/version", s.Version)
}

func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("Index route accessed", map[string]any{
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
	})
	
	response := map[string]any{
		"message": "API Server is running",
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
	}
	s.WriteJSON(w, http.StatusOK, response)
}

// Health check for the API
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	// Health Check will normally not be logged
	response := map[string]any{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}
	s.WriteJSON(w, http.StatusOK, response)
}

// Gives back the current version of the API
func (s *Server) Version(w http.ResponseWriter, r *http.Request) {
	s.logger.Debug("Version info requested")
	
	response := map[string]any{
		"version": "1.0.0",
		"api":     "v1",
	}
	s.WriteJSON(w, http.StatusOK, response)
}

// Writes a JSON response
func (s *Server) WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	response := APIResponse{
		Status: status,
		Data:   data,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to write JSON response", map[string]any{
			"error":  err.Error(),
			"status": status,
		})
		// Fallback if the response fails
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// The same shit in Red
func (s *Server) WriteJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	response := APIResponse{
		Status: status,
		Error:  message,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Failed to write JSON error response", map[string]any{
			"error":   err.Error(),
			"status":  status,
			"message": message,
		})
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// WriteError gives an Error response
func (s *Server) WriteError(w http.ResponseWriter, status int, message string) {
	s.logger.Error("API Error", map[string]any{
		"status":  status,
		"message": message,
	})
	
	response := map[string]any{
		"error":   message,
		"status":  status,
		"time":    time.Now().Format(time.RFC3339),
	}
	s.WriteJSON(w, status, response)
}

// Router-Access

func (s *Server) GetLogger() *logger.Logger {
	return s.logger
}

func (s *Server) SetLogger(l *logger.Logger) {
	s.logger = l
}

func (s *Server) GetRouter() *chi.Mux {
	return s.router
}

func (s *Server) Route(pattern string, fn func(r chi.Router)) {
	s.router.Route(pattern, fn)
}

func (s *Server) Get(pattern string, handlerFn http.HandlerFunc) {
	s.router.Get(pattern, handlerFn)
}

func (s *Server) Post(pattern string, handlerFn http.HandlerFunc) {
	s.router.Post(pattern, handlerFn)
}

func (s *Server) Put(pattern string, handlerFn http.HandlerFunc) {
	s.router.Put(pattern, handlerFn)
}

func (s *Server) Delete(pattern string, handlerFn http.HandlerFunc) {
	s.router.Delete(pattern, handlerFn)
}

func (s *Server) Mount(pattern string, handler http.Handler) {
	s.router.Mount(pattern, handler)
}

// Middleware Methods

func (s *Server) Use(middlewares ...func(http.Handler) http.Handler) {
	s.router.Use(middlewares...)
}

func (s *Server) Group(fn func(r chi.Router)) {
	s.router.Group(fn)
}

// Server-Lifecycle

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30000,
		WriteTimeout: 30000,
	}

	s.logger.Info("Server starting", map[string]any{
		"address":      addr,
		"read_timeout": 30000,
		"write_timeout": 30000,
		"debug":        s.config.Debug,
	})
	return s.httpServer.ListenAndServe()
}

func (s *Server) StartTLS(certFile, keyFile string) error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  30000,
		WriteTimeout: 30000,
	}

	s.logger.Info("Server starting", map[string]any{
		"address":      addr,
		"read_timeout": 30000,
		"write_timeout": 30000,
		"debug":        s.config.Debug,
	})
	return s.httpServer.ListenAndServeTLS(certFile, keyFile)
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	
	s.logger.Info("Server shutdown initiated")
	
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.logger.Error("Server shutdown error", map[string]any{
			"error": err.Error(),
		})
	} else {
		s.logger.Info("Server stopped gracefully")
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) GetAddress() string {
	return fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
}

func (s *Server) IsRunning() bool {
	return s.httpServer != nil
}

var defaultServer *Server

func Init(config *config.Config, log *logger.Logger) *Server {
	defaultServer = NewServer(config, log)
	return defaultServer
}