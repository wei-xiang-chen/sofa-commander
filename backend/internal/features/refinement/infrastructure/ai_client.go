package infrastructure

import (
	"context"
)

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Conversation represents a conversation session
type Conversation struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
}

// AIResponse represents the response from an AI service
type AIResponse struct {
	Content string `json:"content"`
	Error   error  `json:"error,omitempty"`
}

// AIClient defines a generic interface for AI services
type AIClient interface {
	// CreateConversation creates a new conversation session
	CreateConversation(ctx context.Context) (*Conversation, error)

	// AddMessage adds a message to the conversation
	AddMessage(ctx context.Context, conversationID string, role, content string) error

	// GenerateResponse generates a response based on the conversation
	GenerateResponse(ctx context.Context, conversationID string, systemPrompt string) (*AIResponse, error)

	// GetConversation retrieves a conversation by ID
	GetConversation(ctx context.Context, conversationID string) (*Conversation, error)

	// Close closes the client and cleans up resources
	Close() error
}

// AIConfig holds configuration for AI clients
type AIConfig struct {
	Provider string            `json:"provider"` // "openai", "gemini", "claude"
	APIKey   string            `json:"api_key"`
	Model    string            `json:"model"`
	Options  map[string]string `json:"options,omitempty"`
}

// AIClientFactory creates AI clients based on configuration
type AIClientFactory interface {
	CreateClient(config AIConfig) (AIClient, error)
}
