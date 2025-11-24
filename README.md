# cb2utorial - Codebase to Tutorial Generator

Generate tutorial documentation from codebases using LLMs and Restate workflows.

## Architecture

This application uses Restate for durable execution:

1. **cb2utorial service** (port 9082) - Your service that runs the tutorial generation logic
2. **Restate server** (typically port 9070) - The Restate runtime that manages durable execution
3. **CLI client** - Invokes workflows through the Restate server

## Setup

### 1. Start Restate Server

Make sure the Restate server is running:
```bash
restate-server
```

The server should start on port 9070 (default).

### 2. Start cb2utorial Service

```bash
./cb2utorial
```

This starts your service on port 9082 and binds all tutorial generation services.

### 3. Register Services with Restate

Register your service deployment with the Restate server:

```bash
# If you have Restate CLI:
restate deployments register http://localhost:9082

# OR using curl:
curl -X POST http://localhost:9070/deployments \\
  -H "Content-Type: application/json" \\
  -d '{"uri": "http://localhost:9082"}'
```

This tells the Restate server about your service and its handlers.

### 4. Invoke the Workflow

Now you can invoke the workflow through the **Restate server ingress** (port 9070), not directly:

```bash
# Using the CLI (after updating it):
cd cmd/cli
go run main.go --repo /path/to/repo --output ./tutorial

# OR using curl directly through Restate ingress:
curl -X POST http://localhost:9070/TutorialWorkflow/Run \\
  -H "Content-Type: application/json" \\
  -d '{
    "localRepoPath": "/path/to/repo",
    "outputDir": "./tutorial",
    "maxFiles": 100
  }'
```

## Environment Variables

Create a `.env` file:
```
OPENROUTER_API_KEY=your_api_key_here
```

## How It Works

1. **FileReaderService** - Reads and indexes files from the repository
2. **AbstractionAnalyzerService** - Identifies key code abstractions using LLM
3. **RelationshipAnalyzerService** - Analyzes how abstractions relate
4. **ChapterOrdererService** - Determines pedagogical chapter order
5. **ChapterWriterService** - Generates tutorial chapters
6. **FileWriterService** - Writes markdown files to disk
7. **TutorialWorkflow** - Orchestrates the entire process durably

## Troubleshooting

### "Connection reset by peer" error
- Make sure Restate server is running
- Ensure service is registered with Restate
- Use Restate ingress port (9070), not service port (9082) for invocations

### Service binding errors
- Check that `Bind()` uses method chaining (returns `*Restate`, not error)
- Verify all services have proper method signatures
