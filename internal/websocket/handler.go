package websocket

import (
	"ai-gateway/api/gen/agent/v1"
	"ai-gateway/common/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Solve cross-domain problems
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWs(manager *ClientManager, agentClient agentv1.AgentServiceClient, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Log.Error("Failed to upgrade to websocket", zap.Error(err))
		return
	}

	sessionID := uuid.New().String()
	client := &Client{
		Manager:     manager,
		AgentClient: agentClient,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		ID:          sessionID,
	}

	client.Manager.Register <- client

	go client.WritePump()
	go client.ReadPump()
}
