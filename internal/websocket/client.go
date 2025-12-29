package websocket

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/yeliheng/go-ai-gateway/api/gen/agent/v1"
	"github.com/yeliheng/go-ai-gateway/common/logger"
	"github.com/yeliheng/go-ai-gateway/pkg/protocol"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024
)

type Client struct {
	Manager     *ClientManager
	AgentClient agentv1.AgentServiceClient
	Conn        *websocket.Conn
	Send        chan []byte
	ID          string
}

func (c *Client) ReadPump() {
	defer func() {
		c.Manager.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, messageData, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Error("websocket error", zap.Error(err))
			}
			break
		}

		// Parse standard protocol message
		var msg protocol.Message
		if err := json.Unmarshal(messageData, &msg); err != nil {
			logger.Log.Warn("Invalid message format", zap.Error(err))
			c.sendError(400, "Invalid JSON format")
			continue
		}

		logger.Log.Info("Received message", zap.String("client_id", c.ID), zap.String("type", string(msg.Type)))

		switch msg.Type {
		case protocol.TypeChat:
			var payload protocol.ChatPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				c.sendError(400, "Invalid chat payload")
				continue
			}
			c.handleChat(payload)

		case protocol.TypePing:
			c.sendJSON(protocol.Message{Type: protocol.TypePong})

		default:
			c.sendError(404, "Unknown message type")
		}
	}
}

func (c *Client) handleChat(payload protocol.ChatPayload) {
	ctx, cancel := context.WithCancel(context.Background())

	stream, err := c.AgentClient.ChatStream(ctx, &agentv1.ChatRequest{
		Model:   payload.Model,
		Content: payload.Content,
	})

	if err != nil {
		c.sendError(500, err.Error())
		cancel()
		return
	}

	go func() {
		defer cancel()
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				logger.Log.Error("Stream error", zap.Error(err))
				c.sendError(500, "Stream interrupted")
				break
			}

			respPayload := protocol.ChatPayload{
				Content: resp.Content,
				Type:    resp.Type,
				Model:   payload.Model,
			}
			payloadBytes, _ := json.Marshal(respPayload)

			respMsg := protocol.Message{
				Type:    protocol.TypeChat,
				Payload: payloadBytes,
			}

			c.sendJSON(respMsg)
		}
	}()
}

func (c *Client) sendJSON(msg protocol.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Error("Failed to marshal message", zap.Error(err))
		return
	}
	c.Send <- data
}

func (c *Client) sendError(code int, message string) {
	payload := protocol.ErrorPayload{
		Code:    code,
		Message: message,
	}
	b, _ := json.Marshal(payload)
	c.sendJSON(protocol.Message{
		Type:    protocol.TypeError,
		Payload: b,
	})
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
