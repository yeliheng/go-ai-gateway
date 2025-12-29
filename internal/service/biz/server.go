package biz

import (
	"fmt"

	"github.com/yeliheng/go-ai-gateway/api/gen/agent/v1"
	"github.com/yeliheng/go-ai-gateway/api/gen/identity/v1"
	"github.com/yeliheng/go-ai-gateway/common/config"
	"github.com/yeliheng/go-ai-gateway/common/logger"
	"github.com/yeliheng/go-ai-gateway/internal/cache"
	"github.com/yeliheng/go-ai-gateway/internal/handler"
	"github.com/yeliheng/go-ai-gateway/internal/middleware"
	"github.com/yeliheng/go-ai-gateway/internal/websocket"
	"github.com/yeliheng/go-ai-gateway/pkg/telemetry"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewServer() *gin.Engine {
	cache.InitRedis()

	// Init Tracing
	jaegerAddr := config.GlobalConfig.Services.Jaeger.Addr
	if jaegerAddr == "" {
		logger.Log.Fatal("Jaeger address is required but missing")
	}
	shutdown, err := telemetry.InitTracer("gateway", jaegerAddr)
	if err != nil {
		logger.Log.Error("Failed to init tracer", zap.Error(err))
	} else {
		// Handle shutdown gracefully
		_ = shutdown
	}

	// Connect to Identity Service
	identityAddr := config.GlobalConfig.Services.Identity.Addr
	if identityAddr == "" {
		logger.Log.Fatal("Identity service address is required but missing")
	}
	identityConn, err := grpc.NewClient(
		identityAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		logger.Log.Fatal("Failed to connect to Identity Service", zap.Error(err))
	}
	identityClient := identityv1.NewIdentityServiceClient(identityConn)

	// Connect to Agent Service
	agentAddr := config.GlobalConfig.Services.Agent.Addr
	if agentAddr == "" {
		logger.Log.Fatal("Agent service address is required but missing")
	}
	agentConn, err := grpc.NewClient(
		agentAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		logger.Log.Fatal("Failed to connect to Agent Service", zap.Error(err))
	}
	agentClient := agentv1.NewAgentServiceClient(agentConn)

	r := gin.Default()
	r.Use(otelgin.Middleware("gateway"))

	// Handler Injection
	authHandler := handler.NewAuthHandler(identityClient)

	// Auth Routes
	r.Use(middleware.RateLimitMiddleware()) // Global Rate Limit
	r.POST("/register", authHandler.Register)
	r.POST("/login", authHandler.Login)

	// WebSocket Manager
	wsManager := websocket.NewClientManager()
	go wsManager.Run()

	// Routes
	r.GET("/chat", middleware.WebSocketAuthMiddleware(), func(c *gin.Context) {
		websocket.ServeWs(wsManager, agentClient, c)
	})

	r.LoadHTMLFiles("web/index.html", "web/login.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})
	r.GET("/login", func(c *gin.Context) {
		c.HTML(200, "login.html", nil)
	})

	return r
}

func Run() error {
	r := NewServer()
	port := config.GlobalConfig.Services.Biz.Port
	if port == "" {
		logger.Log.Fatal("Biz service port is required but missing")
	}
	return r.Run(fmt.Sprintf(":%s", port))
}
