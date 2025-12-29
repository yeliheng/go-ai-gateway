package provider

import (
	"ai-gateway/common/config"
	"ai-gateway/common/logger"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type OpenAIProvider struct {
	client *http.Client
	cfg    config.OpenAIConfig
}

func NewOpenAIProvider() *OpenAIProvider {
	return &OpenAIProvider{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		cfg: config.GlobalConfig.OpenAI,
	}
}

func (o *OpenAIProvider) Name() string {
	return "openai"
}

func (o *OpenAIProvider) Stream(ctx context.Context, input string) (<-chan Chunk, error) {
	reqBody := ChatCompletionRequest{
		Model: o.cfg.Model,
		Messages: []Message{
			{Role: "system", Content: o.cfg.SystemPrompt},
			{Role: "user", Content: input},
		},
		Stream: true,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", o.cfg.BaseURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+o.cfg.ApiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	outputChan := make(chan Chunk)

	go func() {
		defer close(outputChan)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					logger.Log.Error("Stream read error", zap.Error(err))
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data:") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			data = strings.TrimSpace(data)

			if data == "[DONE]" {
				return
			}

			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				continue
			}

			if len(streamResp.Choices) > 0 {
				delta := streamResp.Choices[0].Delta

				// Thinking
				if delta.ReasoningContent != "" {
					select {
					case outputChan <- Chunk{Content: delta.ReasoningContent, Type: "reasoning"}:
					case <-ctx.Done():
						return
					}
				}

				// Content
				if delta.Content != "" {
					select {
					case outputChan <- Chunk{Content: delta.Content, Type: "text"}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return outputChan, nil
}

type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StreamResponse struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"` // Support reasoning
		} `json:"delta"`
	} `json:"choices"`
}
