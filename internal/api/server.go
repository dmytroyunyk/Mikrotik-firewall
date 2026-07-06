package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dmytroyunyk/mikrotik-defender/internal/metrics"
	"github.com/dmytroyunyk/mikrotik-defender/internal/mikrotik"
	"github.com/dmytroyunyk/mikrotik-defender/internal/storage"
	"github.com/dmytroyunyk/mikrotik-defender/pkg/utils"
)

type Server struct {
	router  *gin.Engine
	server  *http.Server
	db      *storage.DB
	client  *mikrotik.Client
	logger  *utils.Logger
	apikey  string
	metrics *metrics.Metrics
}

func New(
	db *storage.DB,
	client *mikrotik.Client,
	logger *utils.Logger,
	apiKey string,
	m *metrics.Metrics,
) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	router.Use(gin.Recovery())

	s := &Server{
		router:  router,
		db:      db,
		client:  client,
		logger:  logger,
		apikey:  apiKey,
		metrics: m,
	}

	s.registerRoutes()

	return s
}

func (s *Server) registerRoutes() {
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/metrics", gin.WrapH(s.metrics.Handler()))

	api := s.router.Group("/api/v1")
	api.Use(s.authMiddleware())
	{
		api.GET("/stats", s.handleGetStats)

		api.GET("/blocked", s.handleGetBlocked)
		api.DELETE("/blocked/:ip", s.handleUnblock)

		api.GET("/events", s.handleGetEvents)

		api.GET("/attackers", s.handleGetTopAttackers)
	}
}

func (s *Server) Start(port int) error {
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	s.logger.Info("API server started", "port", port)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("API server error: %w", err)
	}
	return nil
}

func (s *Server) Stop() error {
	s.logger.Info("stoping API server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}
