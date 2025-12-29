package provider

import (
	"context"
)

type Chunk struct {
	Content string
	Type    string // "text" or "reasoning"
	Error   error  // Optional error
}

type AIProvider interface {
	Stream(ctx context.Context, input string) (<-chan Chunk, error)
	Name() string
}
