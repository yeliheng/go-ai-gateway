package main

import (
	"ai-gateway/api/gen/agent/v1"
	"ai-gateway/common/config"
	"ai-gateway/common/logger"
	"ai-gateway/internal/provider"
	agentService "ai-gateway/internal/service/agent"
	"ai-gateway/pkg/telemetry"
	"fmt"
	"net"

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

	providers := make(map[string]provider.AIProvider)
	providers["openai"] = provider.NewOpenAIProvider()
	providers["mock"] = provider.NewMockProvider()

	jaegerAddr := config.GlobalConfig.Services.Jaeger.Addr
	if jaegerAddr == "" {
		logger.Log.Fatal("Jaeger address is required but missing")
	}
	shutdown, err := telemetry.InitTracer("agent-service", jaegerAddr)
	if err != nil {
		logger.Log.Error("Failed to init tracer", zap.Error(err))
	} else {
		defer shutdown(nil)
	}

	port := config.GlobalConfig.Services.Agent.Port
	if port == "" {
		logger.Log.Fatal("Agent service port is required but missing")
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		logger.Log.Fatal("Failed to listen", zap.Error(err))
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	agentv1.RegisterAgentServiceServer(s, agentService.NewAgentServer(providers))

	reflection.Register(s)

	logger.Log.Info("Agent Service listening", zap.String("port", port))
	if err := s.Serve(lis); err != nil {
		logger.Log.Fatal("Failed to serve", zap.Error(err))
	}
}
