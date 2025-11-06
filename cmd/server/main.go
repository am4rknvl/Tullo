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
	"github.com/tullo/backend/internal/moderator"
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
	modRepo := repository.NewModerationRepository(db)
	userRepo := repository.NewUserRepository(db)
	convRepo := repository.NewConversationRepository(db)
	msgRepo := repository.NewMessageRepository(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, jwtService)
	convHandler := handlers.NewConversationHandler(convRepo, userRepo, msgRepo)
	msgHandler := handlers.NewMessageHandler(msgRepo, convRepo, redis)

	// Channel & stream repositories and handlers
	chRepo := repository.NewChannelRepository(db)
	streamRepo := repository.NewStreamRepository(db)
	channelHandler := handlers.NewChannelHandler(chRepo, streamRepo, convRepo, userRepo, modRepo)
	// configure local fallback rate/burst using env via config (burst default 10)
	channelChatHandler := handlers.NewChannelChatHandler(chRepo, convRepo, msgRepo, redis, float64(cfg.API.RateLimitMessagesPerSec), 10)

	// Initialize WebSocket hub (only if Redis is available)
	var hub *websocket.Hub
	var wsHandler *websocket.Handler
	if redis != nil {
		hub = websocket.NewHub(redis, convRepo)
		go hub.Run()
		// Ensure TulloBot system user exists
		botUser, err := userRepo.EnsureSystemUser("tullo-bot@tullo.local", "TulloBot")
		if err != nil {
			log.Printf("Warning: failed to ensure TulloBot user: %v", err)
		}

		// Start moderation bot
		bot := moderator.NewBot(redis, convRepo, msgRepo, modRepo, userRepo, botUser.ID)
		go bot.Run()
		wsHandler = websocket.NewHandler(hub, jwtService, msgRepo, convRepo, redis, cfg.CORS.AllowedOrigins)
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
		// Moderation endpoints
		api.POST("/conversations/:id/moderation", convHandler.AddModeration)
		api.DELETE("/conversations/:id/moderation/:user_id", convHandler.RemoveModeration)

		// Message routes
		api.GET("/messages", msgHandler.GetMessages)
		api.POST("/messages", middleware.RateLimitMiddleware(rateLimiter), msgHandler.SendMessage)
		api.PUT("/messages/:id/read", msgHandler.MarkMessageAsRead)

		// WebSocket info (only if Redis is available)
		if wsHandler != nil {
			api.GET("/online-users", wsHandler.GetOnlineUsers)
		}

		// Channel routes
		api.POST("/channels", channelHandler.CreateChannel)
		api.GET("/channels/:slug", channelHandler.GetChannel)
		api.POST("/channels/:slug/start", channelHandler.StartStream)
		api.POST("/channels/:slug/end", channelHandler.EndStream)
		api.GET("/streams", channelHandler.GetActiveStreams)
		api.POST("/channels/:slug/follow", channelHandler.FollowChannel)
		api.DELETE("/channels/:slug/unfollow", channelHandler.UnfollowChannel)
		// channel-level moderator management
		api.POST("/channels/:slug/mods", channelHandler.AssignModerator)
		api.DELETE("/channels/:slug/mods/:user_id", channelHandler.RemoveModerator)
		// ban/unban
		api.POST("/channels/:slug/ban/:user_id", channelHandler.BanUser)
		api.DELETE("/channels/:slug/unban/:user_id", channelHandler.UnbanUser)

		// Channel chat routes
		api.GET("/channels/:slug/chat", channelChatHandler.GetChat)
		api.POST("/channels/:slug/chat", middleware.RateLimitMiddleware(rateLimiter), channelChatHandler.PostChat)
	}

	// Start server
	addr := ":" + cfg.Server.Port
	log.Printf("Starting Tullo server on %s (env: %s)", addr, cfg.Server.Env)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
