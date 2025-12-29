package agent

import (
	"ai-gateway/api/gen/agent/v1"
	"ai-gateway/common/logger"
	"ai-gateway/internal/provider"
	"fmt"

	"go.uber.org/zap"
)

type Server struct {
	agentv1.UnimplementedAgentServiceServer
	providers map[string]provider.AIProvider
}

func NewAgentServer(providers map[string]provider.AIProvider) *Server {
	return &Server{
		providers: providers,
	}
}

func (s *Server) ChatStream(req *agentv1.ChatRequest, stream agentv1.AgentService_ChatStreamServer) error {
	logger.Log.Info("ChatStream request received", zap.String("model", req.Model))
	pName := req.Model
	if pName == "" {
		pName = "mock"
	}

	p, ok := s.providers[pName]
	if !ok {
		logger.Log.Warn("Provider not found", zap.String("provider", pName))
		return fmt.Errorf("provider not found: %s", pName)
	}

	chunkChan, err := p.Stream(stream.Context(), req.Content)
	if err != nil {
		logger.Log.Error("Provider stream error", zap.Error(err))
		return err
	}

	for chunk := range chunkChan {
		if chunk.Error != nil {
			logger.Log.Error("Chunk error in stream", zap.Error(chunk.Error))
			return chunk.Error
		}

		resp := &agentv1.ChatResponse{
			Content: chunk.Content,
			Type:    chunk.Type,
		}

		if err := stream.Send(resp); err != nil {
			logger.Log.Error("Failed to send stream response", zap.Error(err))
			return err
		}
	}

	logger.Log.Info("ChatStream completed successfully")
	return nil
}
