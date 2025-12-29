package main

import (
	"fmt"
	"net"

	"github.com/yeliheng/go-ai-gateway/api/gen/identity/v1"
	"github.com/yeliheng/go-ai-gateway/common/config"
	"github.com/yeliheng/go-ai-gateway/common/logger"
	"github.com/yeliheng/go-ai-gateway/internal/cache"
	"github.com/yeliheng/go-ai-gateway/internal/database"
	identityService "github.com/yeliheng/go-ai-gateway/internal/service/identity"
	"github.com/yeliheng/go-ai-gateway/pkg/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize Logger
	logger.InitLogger()
	defer logger.Log.Sync()

	if err := config.LoadConfig(); err != nil {
		logger.Log.Fatal("Failed to load config", zap.Error(err))
	}

	// Connect to Database
	database.InitDB()
	cache.InitRedis()

	jaegerAddr := config.GlobalConfig.Services.Jaeger.Addr
	if jaegerAddr == "" {
		logger.Log.Fatal("Jaeger address is required but missing")
	}
	shutdown, err := telemetry.InitTracer("identity-service", jaegerAddr)
	if err != nil {
		logger.Log.Error("Failed to init tracer", zap.Error(err))
	} else {
		defer shutdown(nil)
	}

	port := config.GlobalConfig.Services.Identity.Port
	if port == "" {
		logger.Log.Fatal("Identity service port is required but missing")
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Log.Fatal("Failed to listen", zap.Error(err))
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	identityv1.RegisterIdentityServiceServer(s, identityService.NewIdentityServer())

	// Enable reflection for debugging
	reflection.Register(s)

	logger.Log.Info("Identity Service listening", zap.String("port", port))
	if err := s.Serve(lis); err != nil {
		logger.Log.Fatal("Failed to serve", zap.Error(err))
	}
}
