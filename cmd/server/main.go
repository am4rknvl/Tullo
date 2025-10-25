package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/tullo/backend/config"
	"github.com/tullo/backend/internal/auth"
	"github.com/tullo/backend/internal/cache"
	"github.com/tullo/backend/internal/database"
	"github.com/tullo/backend/internal/handlers"
	"github.com/tullo/backend/internal/middleware"
	"github.com/tullo/backend/internal/repository"
	"github.com/tullo/backend/internal/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := database.NewPostgresDB(cfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	log.Println("Running database migrations...")
	if err := database.RunMigrations(db.DB); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations completed successfully")

	// Connect to Redis
	redis, err := cache.NewRedisClient(cfg.GetRedisAddr(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		log.Println("Running without Redis - real-time features will be limited")
		redis = nil
	} else {
		defer redis.Close()
	}

	// Initialize services
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpiryHours)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	convRepo := repository.NewConversationRepository(db)
	msgRepo := repository.NewMessageRepository(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, jwtService)
	convHandler := handlers.NewConversationHandler(convRepo, userRepo, msgRepo)
	msgHandler := handlers.NewMessageHandler(msgRepo, convRepo, redis)

	// Initialize WebSocket hub (only if Redis is available)
	var hub *websocket.Hub
	var wsHandler *websocket.Handler
	if redis != nil {
		hub = websocket.NewHub(redis)
		go hub.Run()
		wsHandler = websocket.NewHandler(hub, jwtService, msgRepo, convRepo, redis)
	}

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(cfg.API.RateLimitMessagesPerSec)
	rateLimiter.Cleanup()

	// Setup Gin router
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware
	router.Use(middleware.CORSMiddleware(cfg.CORS.AllowedOrigins))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public routes
	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
	}

	// WebSocket endpoint (only if Redis is available)
	if wsHandler != nil {
		router.GET("/ws", wsHandler.HandleWebSocket)
	}

	// Protected routes
	api := router.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtService))
	{
		// User routes
		api.GET("/me", authHandler.GetMe)

		// Conversation routes
		api.GET("/conversations", convHandler.GetConversations)
		api.POST("/conversations", convHandler.CreateConversation)
		api.GET("/conversations/:id", convHandler.GetConversation)
		api.POST("/conversations/:id/members", convHandler.AddMembers)
		api.DELETE("/conversations/:id/members/:user_id", convHandler.RemoveMember)

		// Message routes
		api.GET("/messages", msgHandler.GetMessages)
		api.POST("/messages", middleware.RateLimitMiddleware(rateLimiter), msgHandler.SendMessage)
		api.PUT("/messages/:id/read", msgHandler.MarkMessageAsRead)

		// WebSocket info (only if Redis is available)
		if wsHandler != nil {
			api.GET("/online-users", wsHandler.GetOnlineUsers)
		}
	}

	// Start server
	addr := ":" + cfg.Server.Port
	log.Printf("Starting Tullo server on %s (env: %s)", addr, cfg.Server.Env)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
