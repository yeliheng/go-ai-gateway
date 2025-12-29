package main

import (
	"ai-gateway/common/config"
	"ai-gateway/common/logger"
	"ai-gateway/internal/service/biz"

	"go.uber.org/zap"
)

func main() {
	// Initialize Logger
	logger.InitLogger()
	defer logger.Log.Sync()

	if err := config.LoadConfig(); err != nil {
		logger.Log.Fatal("Failed to load config", zap.Error(err))
	}
	logger.Log.Info("Configuration loaded", zap.String("app_name", config.GlobalConfig.App.Name))

	// Start Server
	logger.Log.Info("Starting biz", zap.String("port", config.GlobalConfig.App.Port))
	if err := biz.Run(); err != nil {
		logger.Log.Fatal("Server failed", zap.Error(err))
	}
}
