package api

import (
	"log/slog"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	logger *slog.Logger
	Addr   string
	Router *gin.Engine
}

func NewServer(addr string, logger *slog.Logger) *Server {
	return &Server{
		Addr:   addr,
		Router: gin.Default(),
		logger: logger,
	}
}

func (s *Server) SetupDefaultConfig() {
	s.Router.Use(cors.New(cors.Config{
		// AllowOrigins:     []string{"http://localhost:3000"}, // React
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		AllowAllOrigins:  true, // Allow all origins for testing purposes
	}))
}

func (s *Server) Run() error {
	return s.Router.Run(s.Addr)
}
