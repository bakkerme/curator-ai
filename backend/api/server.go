package api

import (
	"net/http"

	"curator/config"
	"curator/storage"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	config *config.Config
	db     *storage.BadgerDB
	echo   *echo.Echo
}

func NewServer(cfg *config.Config, db *storage.BadgerDB) *Server {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	server := &Server{
		config: cfg,
		db:     db,
		echo:   e,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// Enable CORS for frontend
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
	}))

	// API routes
	api := s.echo.Group("/api/v1")
	api.GET("/health", s.handleHealth)
	api.GET("/status", s.handleStatus)

	// Pipeline routes
	pipeline := api.Group("/pipeline")
	pipeline.GET("/config", s.handleGetPipelineConfig)
	pipeline.POST("/config", s.handleUpdatePipelineConfig)
	pipeline.GET("/status", s.handleGetPipelineStatus)
	pipeline.POST("/run", s.handleRunPipeline)
}

func (s *Server) Start(addr string) error {
	return s.echo.Start(addr)
}

func (s *Server) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"service": "curator",
	})
}

func (s *Server) handleStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "running",
		"version": "0.1.0",
		"config": map[string]interface{}{
			"llm_provider": s.config.LLM.Provider,
			"llm_endpoint": s.config.LLM.Endpoint,
		},
	})
}

func (s *Server) handleGetPipelineConfig(c echo.Context) error {
	// TODO: Implement pipeline configuration retrieval
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Pipeline configuration endpoint - not implemented yet",
	})
}

func (s *Server) handleUpdatePipelineConfig(c echo.Context) error {
	// TODO: Implement pipeline configuration update
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Pipeline configuration update endpoint - not implemented yet",
	})
}

func (s *Server) handleGetPipelineStatus(c echo.Context) error {
	// TODO: Implement pipeline status retrieval
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "idle",
		"last_run": nil,
		"next_run": nil,
	})
}

func (s *Server) handleRunPipeline(c echo.Context) error {
	// TODO: Implement pipeline execution
	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message": "Pipeline execution started",
		"run_id":  "placeholder-run-id",
	})
}
