package provider

import (
	"context"
	"math/rand"
	"time"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Name() string {
	return "mock"
}

func (p *MockProvider) Stream(ctx context.Context, input string) (<-chan Chunk, error) {
	outputChan := make(chan Chunk)

	go func() {
		defer close(outputChan)

		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}

		// Simulate reasoning phase
		reasoning := "Hmm... let me think about " + input + "..."
		for _, r := range reasoning {
			select {
			case <-ctx.Done():
				return
			case outputChan <- Chunk{Content: string(r), Type: "reasoning"}:
				time.Sleep(time.Duration(rand.Intn(30)+10) * time.Millisecond)
			}
		}

		// Simulate content phase
		response := "\nOkay, here is the answer: [Mock Output]"
		for _, r := range response {
			select {
			case <-ctx.Done():
				return
			case outputChan <- Chunk{Content: string(r), Type: "text"}:
				time.Sleep(time.Duration(rand.Intn(50)+30) * time.Millisecond)
			}
		}
	}()

	return outputChan, nil
}
