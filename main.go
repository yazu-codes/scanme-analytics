package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/yazu-codes/scanme-analytics.git/src/api"
	"github.com/yazu-codes/scanme-analytics.git/src/api/handlers"
	"github.com/yazu-codes/scanme-analytics.git/src/api/middleware"
	db "github.com/yazu-codes/scanme-analytics.git/src/database"
	"github.com/yazu-codes/scanme-analytics.git/src/repositories"
	"github.com/yazu-codes/scanme-analytics.git/src/util"
	"golang.org/x/time/rate"
)

func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)
	logger = logger.With(slog.String("component", "analytics_service"))

	var config *util.ConfigReader = util.NewConfigReader()
	config.Setup()

	db := db.NewDB(config.Admin.Email, config.Admin.Password, config.Db.DSN(), logger)
	db.Connect()
	db.AutoMigrate()

	server := api.NewServer(config.Server.ConstructUrl(), logger)
	server.SetupDefaultConfig()

	// Initialize services and repositories
	eventRepo := repositories.NewEventsRepository(db)
	eventHandler := handlers.NewEventHandler(eventRepo)

	lim := middleware.NewLimiter(middleware.RateLimitConfig{
		Secret:    []byte(os.Getenv("IP_HASH_SECRET")), // 32+ random bytes
		Rate:      rate.Every(500 * time.Millisecond),
		Burst:     20,
		SkipPaths: []string{"/healthz"},
	})
	defer lim.Close()

	// Protected routes
	rateLimited := server.Router.Group("/events")
	rateLimited.Use(middleware.EventVerification(), lim.IpRateLimitMiddleware())
	{
		rateLimited.POST("/", eventHandler.RecordEvent)
		rateLimited.GET("/", middleware.AuthMiddleware(config.JWTConfig.Secret), middleware.RequireRole("admin"), eventHandler.GetEvents)
	}

	server.Run()
}
