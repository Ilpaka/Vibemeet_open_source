// Vibemeet - a self-hostable, WebRTC-based video conferencing server.
// This is the process entrypoint: it parses subcommands, wires configuration
// and dependencies, applies pending database migrations, and runs the HTTP
// router with graceful shutdown.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vibemeet/internal/config"
	"vibemeet/internal/handler"
	"vibemeet/internal/middleware"
	"vibemeet/internal/migration"
	"vibemeet/internal/repository"
	"vibemeet/internal/service"
	"vibemeet/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			runMigrate(os.Args[2:])
			return
		case "-h", "--help", "help":
			printUsage()
			return
		}
	}
	runServer()
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Vibemeet server.

Usage:
  vibemeet                       run the HTTP server (default)
  vibemeet migrate up            apply all pending migrations
  vibemeet migrate up-by-one     apply the next migration
  vibemeet migrate down          roll back the most recent migration
  vibemeet migrate redo          roll back and re-apply the latest migration
  vibemeet migrate status        show applied/pending state of every migration
  vibemeet migrate version       print the current schema version

Configuration is read from the environment; see env.sample for the full list.
`)
}

func runServer() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	appLogger := logger.New(cfg.Log.Level)

	dbPool, err := pgxpool.New(context.Background(), cfg.Database.DSN)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", "error", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		appLogger.Fatal("Failed to ping database", "error", err)
	}
	appLogger.Info("Database connection established")

	if cfg.Database.MigrateOnBoot {
		if err := applyMigrations(dbPool); err != nil {
			appLogger.Fatal("Database migrations failed", "error", err)
		}
		appLogger.Info("Database migrations up to date")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer func() { _ = rdb.Close() }()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		appLogger.Fatal("Failed to connect to Redis", "error", err)
	}
	appLogger.Info("Redis connection established")

	repos := repository.NewRepositories(dbPool, rdb, appLogger)
	services := service.NewServices(repos, cfg, appLogger)

	authMiddleware := middleware.NewAuthMiddleware(services.Auth, appLogger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(services.RateLimit, appLogger)
	participantMiddleware := middleware.ParticipantMiddleware()

	handlers := handler.NewHandlers(services, repos, cfg, appLogger)
	router := setupRouter(handlers, authMiddleware, rateLimitMiddleware, participantMiddleware, cfg)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		appLogger.Info("Starting server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown", "error", err)
	}

	appLogger.Info("Server exited")
}

// applyMigrations runs every pending goose migration against the pool.
func applyMigrations(pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	return migration.Apply(ctx, db)
}

func runMigrate(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: vibemeet migrate <up|up-by-one|down|redo|status|version>")
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := sql.Open("pgx", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := args[0]
	switch cmd {
	case "up":
		err = migration.Apply(ctx, db)
	case "up-by-one":
		err = migration.ApplyOne(ctx, db)
	case "down":
		err = migration.Down(ctx, db)
	case "redo":
		err = migration.Redo(ctx, db)
	case "status":
		err = migration.Status(ctx, db)
	case "version":
		v, vErr := migration.Version(ctx, db)
		if vErr != nil {
			log.Fatalf("migrate version: %v", vErr)
		}
		fmt.Printf("schema version: %d\n", v)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown migrate command: %s\n", cmd)
		os.Exit(2)
	}

	if err != nil {
		log.Fatalf("migrate %s: %v", cmd, err)
	}
}

func setupRouter(
	handlers *handler.Handlers,
	authMiddleware *middleware.AuthMiddleware,
	rateLimitMiddleware *middleware.RateLimitMiddleware,
	participantMiddleware gin.HandlerFunc,
	cfg *config.Config,
) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.ErrorHandler())

	router.GET("/health", handlers.Health.Check)
	router.GET("/server-info", handlers.Health.ServerInfo)

	v1 := router.Group("/api/v1")
	{
		// --- Public auth endpoints ------------------------------------------
		auth := v1.Group("/auth")
		{
			auth.POST("/register", rateLimitMiddleware.Limit(), handlers.Auth.Register)
			auth.POST("/login", rateLimitMiddleware.Limit(), handlers.Auth.Login)
			auth.POST("/refresh", handlers.Auth.RefreshToken)
		}

		// --- Anonymous endpoints (no auth; identified by participant_id) ----
		anonymous := v1.Group("")
		anonymous.Use(participantMiddleware)
		{
			if handlers.AnonymousRoom != nil {
				anonymous.POST("/rooms", handlers.AnonymousRoom.Create)
				anonymous.GET("/rooms/:id", handlers.AnonymousRoom.GetByID)
				anonymous.POST("/rooms/:id/join", handlers.AnonymousRoom.Join)
				anonymous.POST("/rooms/:id/leave", handlers.AnonymousRoom.Leave)
				anonymous.GET("/rooms/:id/participants", handlers.AnonymousRoom.GetParticipants)
			}
			if handlers.AnonymousMedia != nil {
				anonymous.POST("/rooms/:id/media/token", handlers.AnonymousMedia.GetToken)
			}
			if handlers.AnonymousChat != nil {
				anonymous.GET("/rooms/:id/chat/messages", handlers.AnonymousChat.GetMessages)
				anonymous.POST("/rooms/:id/chat/messages", handlers.AnonymousChat.SendMessage)
				anonymous.DELETE("/rooms/:id/chat/messages/:messageId", handlers.AnonymousChat.DeleteMessage)
			}
		}

		// --- Authenticated user features ------------------------------------
		protected := v1.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			users := protected.Group("/users")
			{
				users.GET("/me", handlers.User.GetMe)
				users.PUT("/me", handlers.User.UpdateMe)
				users.GET("/me/settings", handlers.User.GetSettings)
				users.PUT("/me/settings", handlers.User.UpdateSettings)
			}

			rooms := protected.Group("/rooms")
			{
				rooms.GET("", handlers.Room.List)
				rooms.PUT("/:id", handlers.Room.Update)
				rooms.DELETE("/:id", handlers.Room.Delete)
				rooms.POST("/:id/invite", handlers.Room.CreateInvite)
			}

			stats := protected.Group("/rooms/:id/stats")
			{
				stats.GET("", handlers.Stats.GetRoomStats)
				stats.GET("/participants/:participantId", handlers.Stats.GetParticipantStats)
			}
		}
	}

	// --- Server-side screen sharing (Pion WebRTC) ---------------------------
	screenShare := router.Group("/screen-share")
	{
		screenShare.POST("/offer", handlers.ScreenShare.HandleOffer)
		screenShare.POST("/ice/:id", handlers.ScreenShare.HandleICE)
		screenShare.GET("/ice/:id", handlers.ScreenShare.GetICE)
		screenShare.POST("/hangup/:id", handlers.ScreenShare.HandleHangup)
		screenShare.GET("/", handlers.ScreenShare.ServeHTML)
	}

	return router
}
