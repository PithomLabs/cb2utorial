# Phase 3: LLM Client Wrapper

## Goal
Create a reusable LLM client using go-openrouter library.

## Decisions Made

**Package Import**: `github.com/revrost/go-openrouter` (note: lowercase in import)

**API Design**:
- `NewClient()` - Initialize from environment variables
- `CallLLM(ctx, prompt, systemPrompt)` - Simple text completion
- Return raw string response for services to parse

**Configuration (from .env)**:
- `OPENROUTER_API_KEY` - API authentication
- `LLM_MODEL` - Model selection (e.g., "openai/gpt-4", "anthropic/claude-3.5")

**Error Handling**:
- Return errors to caller
- Let Restate handle retries at service level
- No internal retry logic (keep it simple)

## Why OpenRouter

**Advantages**:
- Single API for multiple providers (GPT-4, Claude, DeepSeek, etc.)
- Cost-effective (can use free models for testing)
- No vendor lock-in
- Consistent API regardless of model

## API Usage Pattern

Based on go-openrouter docs:
```go
client := openrouter.NewClient(apiKey)
resp, err := client.CreateChatCompletion(ctx, 
    openrouter.ChatCompletionRequest{
        Model: model,
        Messages: []openrouter.ChatCompletionMessage{
            openrouter.UserMessage(prompt),
        },
    })
content := resp.Choices[0].Message.Content
```

## Design Choice: Simple Text Interface

Services will parse YAML/structured output themselves. LLM client just returns raw text.
- Separation of concerns
- Easier to debug
- Flexible for different response formats

## Next Steps
Phase 4 will implement file utilities using this LLM client.
