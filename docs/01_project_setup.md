# Phase 1: Project Setup

## Goal
Initialize Go project structure with dependencies and configuration.

## Decisions Made

**Module Name**: `github.com/yourusername/cb2utorial`
- Generic placeholder, user can change later

**Dependencies**:
- `github.com/pithomlabs/rea` - Restate framework wrapper
- `github.com/restatedev/sdk-go` - Core Restate SDK
- `github.com/reVrost/go-openrouter` - LLM client (OpenRouter API)
- `github.com/gobwas/glob` - File pattern matching
- `gopkg.in/yaml.v3` - YAML parsing for LLM responses
- `github.com/joho/godotenv` - .env file loading

**Directory Structure**:
```
cb2utorial/
├── docs/           # Implementation thought-process docs
├── types/          # Shared type definitions
├── llm/            # LLM client wrapper
├── utils/          # File utilities (walker, writer)
├── services/       # Restate services (data plane)
├── workflow/       # Workflow orchestration (control plane)
├── cmd/generate/   # CLI client
├── go.mod
├── go.sum
├── .env.example
└── main.go
```

**Environment Variables** (.env):
- `OPENROUTER_API_KEY` - API key for OpenRouter
- `LLM_MODEL` - Model to use (e.g., "openai/gpt-4")
- `MAX_FILE_SIZE` - Max file size in bytes
- `MAX_FILES` - Max files to process

## Why These Choices

1. **Rea Framework**: Provides high-level abstractions over Restate SDK
2. **OpenRouter**: Single API for multiple LLM providers (GPT-4, Claude, etc.)
3. **Glob Matching**: Efficient file pattern filtering
4. **YAML**: LLM outputs structured data easily in YAML
5. **Godotenv**: Standard Go library for .env files

## Next Steps
- Create type definitions in Phase 2
- Implement LLM client wrapper in Phase 3
