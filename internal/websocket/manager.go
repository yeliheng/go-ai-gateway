package websocket

import (
	"ai-gateway/common/logger"
	"sync"

	"go.uber.org/zap"
)

type ClientManager struct {
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

func (manager *ClientManager) Run() {
	for {
		select {
		case client := <-manager.Register:
			manager.mu.Lock()
			manager.Clients[client] = true
			manager.mu.Unlock()
			logger.Log.Info("Client registered", zap.String("id", client.ID))

		case client := <-manager.Unregister:
			manager.mu.Lock()
			if _, ok := manager.Clients[client]; ok {
				delete(manager.Clients, client)
				close(client.Send)
				logger.Log.Info("Client unregistered", zap.String("id", client.ID))
			}
			manager.mu.Unlock()

		case message := <-manager.Broadcast:
			logger.Log.Debug("Broadcast message received", zap.String("msg", string(message)))
		}
	}
}
