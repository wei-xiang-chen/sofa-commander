package infrastructure

import (
	"context"
	"fmt"
	"os"
	"time"

	openai "github.com/sashabaranov/go-openai"
	// "sofa-commander/backend/internal/features/refinement/domain" // Not directly used here, but might be needed for other functions later
)

// OpenAIClient defines the interface for an OpenAI client using Assistants API.
type OpenAIClient interface {
	GetOrCreateAssistant(name, instructions, model string) (string, error)
	CreateThread() (string, error)
	AddMessageToThread(threadID, content string) error
	RunAssistant(threadID, assistantID string) error
	GetAssistantResponse(threadID string) ([]openai.Message, error)
}

// openAIClient is the implementation of OpenAIClient.
type openAIClient struct {
	client *openai.Client
	// Store assistant ID in memory for now, could be persisted later
	assistantID string
}

// NewOpenAIClient creates a new OpenAI client, requires OPENAI_API_KEY env var.
func NewOpenAIClient() (OpenAIClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}
	client := openai.NewClient(apiKey)
	return &openAIClient{client: client}, nil
}

// GetOrCreateAssistant creates an assistant if it doesn't exist, or retrieves it.
func (c *openAIClient) GetOrCreateAssistant(name, instructions, model string) (string, error) {
	if c.assistantID != "" {
		return c.assistantID, nil // Already created/retrieved in this session
	}

	// List assistants (paginated, but we just get the first page)
	assistantsList, err := c.client.ListAssistants(context.Background(), nil, nil, nil, nil)
	if err != nil {
		fmt.Printf("[OpenAI] ListAssistants error: %+v\n", err)
		return "", fmt.Errorf("failed to list assistants: %w", err)
	}

	for _, asst := range assistantsList.Assistants {
		if asst.Name != nil && *asst.Name == name {
			c.assistantID = asst.ID
			return asst.ID, nil
		}
	}

	// Assistant not found, create a new one
	fmt.Printf("Creating Assistant with Name: %s, Instructions: %s, Model: %s\n", name, instructions, model)
	newAssistant, err := c.client.CreateAssistant(context.Background(), openai.AssistantRequest{
		Name:         &name,
		Instructions: &instructions,
		Model:        model,
	})
	if err != nil {
		fmt.Printf("[OpenAI] CreateAssistant error: %+v\n", err)
		return "", fmt.Errorf("failed to create assistant: %w", err)
	}
	c.assistantID = newAssistant.ID
	return newAssistant.ID, nil
}

// CreateThread creates a new conversation thread.
func (c *openAIClient) CreateThread() (string, error) {
	fmt.Println("Creating new thread...")
	thread, err := c.client.CreateThread(context.Background(), openai.ThreadRequest{})
	if err != nil {
		fmt.Printf("[OpenAI] CreateThread error: %+v\n", err)
		return "", fmt.Errorf("failed to create thread: %w", err)
	}
	return thread.ID, nil
}

// AddMessageToThread adds a user message to a specific thread.
func (c *openAIClient) AddMessageToThread(threadID, content string) error {
	fmt.Printf("Adding message to thread %s: %s\n", threadID, content)
	_, err := c.client.CreateMessage(context.Background(), threadID, openai.MessageRequest{
		Role:    "user",
		Content: content,
	})

	if err != nil {
		fmt.Printf("[OpenAI] CreateMessage error: %+v\n", err)
		return fmt.Errorf("failed to add message to thread: %w", err)
	}
	return nil
}

// RunAssistant creates a run on a thread and polls for its completion.
func (c *openAIClient) RunAssistant(threadID, assistantID string) error {
	fmt.Printf("Running assistant %s on thread %s\n", assistantID, threadID)
	run, err := c.client.CreateRun(context.Background(), threadID, openai.RunRequest{
		AssistantID: assistantID,
	})

	if err != nil {
		fmt.Printf("[OpenAI] CreateRun error: %+v\n", err)
		return fmt.Errorf("failed to create run: %w", err)
	}

	// Poll for run completion
	for run.Status != openai.RunStatusCompleted && run.Status != openai.RunStatusFailed && run.Status != openai.RunStatusCancelled && run.Status != openai.RunStatusExpired {
		time.Sleep(1 * time.Second) // Poll every second
		run, err = c.client.RetrieveRun(context.Background(), threadID, run.ID)
		if err != nil {
			fmt.Printf("[OpenAI] RetrieveRun error: %+v\n", err)
			return fmt.Errorf("failed to retrieve run status: %w", err)
		}
	}

	if run.Status != openai.RunStatusCompleted {
		return fmt.Errorf("run did not complete successfully, status: %s", run.Status)
	}
	return nil
}

// GetAssistantResponse retrieves the latest assistant message from a thread.
func (c *openAIClient) GetAssistantResponse(threadID string) ([]openai.Message, error) {
	messages, err := c.client.ListMessage(context.Background(), threadID, nil, nil, nil, nil, nil)
	if err != nil {
		fmt.Printf("[OpenAI] ListMessage error: %+v\n", err)
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	var assistantMessages []openai.Message
	for _, msg := range messages.Messages {
		if msg.Role == "assistant" {
			assistantMessages = append(assistantMessages, msg)
		}
	}
	// Messages are returned in reverse chronological order, so reverse them to get latest first
	for i, j := 0, len(assistantMessages)-1; i < j; i, j = i+1, j-1 {
		assistantMessages[i], assistantMessages[j] = assistantMessages[j], assistantMessages[i]
	}

	return assistantMessages, nil
}
