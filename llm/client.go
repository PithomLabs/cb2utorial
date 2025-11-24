package llm

import (
	"context"
	"fmt"
	"os"

	openrouter "github.com/revrost/go-openrouter"
)

// Client wraps OpenRouter client for LLM interactions
type Client struct {
	client *openrouter.Client
	model  string
}

// NewClient creates a new LLM client from environment variables
// Requires: OPENROUTER_API_KEY and LLM_MODEL
func NewClient() (*Client, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY environment variable not set")
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "openai/gpt-4" // Default model
	}

	client := openrouter.NewClient(apiKey)

	return &Client{
		client: client,
		model:  model,
	}, nil
}

// CallLLM sends a prompt to the LLM and returns the text response
// systemPrompt is optional (can be empty string)
func (c *Client) CallLLM(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	messages := []openrouter.ChatCompletionMessage{}

	// Add system message if provided
	if systemPrompt != "" {
		messages = append(messages, openrouter.ChatCompletionMessage{
			Role:    openrouter.ChatMessageRoleSystem,
			Content: openrouter.Content{Text: systemPrompt},
		})
	}

	// Add user prompt
	messages = append(messages, openrouter.UserMessage(prompt))

	// Create chat completion request
	req := openrouter.ChatCompletionRequest{
		Model:    c.model,
		Messages: messages,
	}

	// Call OpenRouter API
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OpenRouter API error: %w", err)
	}

	// Extract response text
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from LLM")
	}

	return resp.Choices[0].Message.Content.Text, nil
}
