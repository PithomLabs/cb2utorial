# Go Implementation Plan: Codebase Tutorial Generator using Rea Framework

## Overview

Port PocketFlow-Tutorial-Codebase-Knowledge to Go using the **rea framework** (Restate Go SDK wrapper). This implementation will read a local GitHub repository folder and generate a series of sequentially-named markdown tutorial files.

## Simplified Requirements

**Input:**
- Local directory path to a Git repository
- Output directory for generated tutorial files

**Output:**
- Sequential markdown files: `01_abstraction_name.md`, `02_abstraction_name.md`, etc.
- No index.md (simplified - just individual chapter files)

## Architecture: Rea Framework Mapping

### Control Plane vs Data Plane Design

```
┌─────────────────────────────────────────────────┐
│          CONTROL PLANE (Orchestration)          │
│                                                 │
│  TutorialWorkflow (restate.WorkflowContext)     │
│    - Orchestrates entire pipeline               │
│    - Manages sequential execution               │
│    - Handles errors with Saga pattern           │
│    - Stores pipeline state                      │
└────────────┬────────────────────────────────────┘
             │
             │ Invokes via ServiceClient
             ▼
┌─────────────────────────────────────────────────┐
│           DATA PLANE (Services)                 │
│                                                 │
│  1. FileReaderService                           │
│  2. AbstractionAnalyzerService                  │
│  3. RelationshipAnalyzerService                 │
│  4. ChapterOrdererService                       │
│  5. ChapterWriterService                        │
│  6. FileWriterService                           │
└─────────────────────────────────────────────────┘
```

### Why This Design?

- **Workflow as Orchestrator**: Single workflow coordinates the entire pipeline, providing durable execution and automatic retries
- **Services as Workers**: Each stage is a stateless service that can be tested independently
- **Saga for Cleanup**: If any stage fails, compensate previously written files
- **Type Safety**: Strongly typed service contracts (vs Python's dynamic typing)

## Component Design

### 1. TutorialWorkflow (Control Plane)

**Type**: `restate.WorkflowContext` (Workflow Service)

**Purpose**: Orchestrate the 5-stage pipeline with durable state

**Input**:
```go
type TutorialWorkflowInput struct {
    LocalRepoPath string
    OutputDir     string
    MaxFiles      int  // Limit files to process
}
```

**State**:
```go
type TutorialState struct {
    Files          []FileContent
    Abstractions   []Abstraction
    Relationships  RelationshipData
    ChapterOrder   []int
    ChaptersWritten []string  // Paths for cleanup
}
```

**Flow**:
1. Initialize Saga for compensation
2. Call `FileReaderService` → Get files
3. Call `AbstractionAnalyzerService` → Get abstractions
4. Call `RelationshipAnalyzerService` → Get relationships
5. Call `ChapterOrdererService` → Get chapter order
6. **Loop** through ordered chapters:
   - Call `ChapterWriterService` for each (parallel via `RequestFuture`)
7. Call `FileWriterService` to write markdown files
8. On error: Saga compensates (deletes written files)

**Saga Pattern**:
```go
saga := framework.NewSaga(ctx, "tutorial-generation", nil)
defer saga.CompensateIfNeeded(&err)

// Register cleanup
saga.Register("write_files", func(rc restate.RunContext, payload []byte) error {
    return deleteGeneratedFiles(outputDir)
})
```

---

### 2. FileReaderService (Data Plane)

**Type**: `restate.Context` (Stateless Service)

**Handler**: `ReadFiles`

**Input**:
```go
type ReadFilesInput struct {
    RepoPath        string
    IncludePatterns []string  // e.g., ["*.go", "*.py"]
    ExcludePatterns []string  // e.g., ["*_test.go", "vendor/*"]
    MaxFileSize     int64
    MaxFiles        int
}
```

**Output**:
```go
type FileContent struct {
    Index   int
    Path    string
    Content string
}

type ReadFilesOutput struct {
    Files []FileContent
}
```

**Logic**:
- Use Go's `filepath.Walk` to traverse directory
- Apply glob pattern matching (use `github.com/gobwas/glob`)
- Filter by file size
- Limit total files processed
- Return indexed list

**Dependencies**:
- Standard library: `filepath`, `os`, `io`
- Third-party: `github.com/gobwas/glob` for pattern matching

---

### 3. AbstractionAnalyzerService (Data Plane)

**Type**: `restate.Context` (Stateless Service)

**Handler**: `AnalyzeAbstractions`

**Input**:
```go
type AnalyzeAbstractionsInput struct {
    Files       []FileContent
    ProjectName string
    MaxAbstractions int  // Default: 10
}
```

**Output**:
```go
type Abstraction struct {
    Index       int
    Name        string
    Description string
    FileIndices []int  // References to FileContent.Index
}

type AnalyzeAbstractionsOutput struct {
    Abstractions []Abstraction
}
```

**Logic**:
- Format files as numbered context for LLM
- Construct prompt requesting YAML output
- Call LLM via `framework.Run()` wrapper (for side-effect logging)
- Parse YAML response
- Validate:
  - Max abstractions limit
  - File indices are within bounds
  - No duplicate indices
- Return structured abstractions

**LLM Integration**:
```go
// Use Run wrapper for side effects
response, err := framework.Run(ctx, "call_llm", func(rc restate.RunContext) (string, error) {
    return callLLM(prompt)
})
```

**Dependencies**:
- LLM client library (e.g., `github.com/sashabaranov/go-openai` or Gemini SDK)
- YAML parser: `gopkg.in/yaml.v3`

---

### 4. RelationshipAnalyzerService (Data Plane)

**Type**: `restate.Context` (Stateless Service)

**Handler**: `AnalyzeRelationships`

**Input**:
```go
type AnalyzeRelationshipsInput struct {
    Abstractions []Abstraction
    Files        []FileContent
    ProjectName  string
}
```

**Output**:
```go
type Relationship struct {
    FromIndex int     // Source abstraction index
    ToIndex   int     // Target abstraction index
    Label     string  // Relationship description
}

type RelationshipData struct {
    Summary  string
    Details  []Relationship
}
```

**Logic**:
- Format abstractions with indices for LLM
- Include relevant file snippets
- Request project summary + relationship list in YAML
- Validate:
  - FromIndex/ToIndex within abstraction bounds
  - No self-references
  - Label is non-empty
- Return structured relationships

---

### 5. ChapterOrdererService (Data Plane)

**Type**: `restate.Context` (Stateless Service)

**Handler**: `OrderChapters`

**Input**:
```go
type OrderChaptersInput struct {
    Abstractions  []Abstraction
    Relationships RelationshipData
    ProjectName   string
}
```

**Output**:
```go
type OrderChaptersOutput struct {
    OrderedIndices []int  // Abstraction indices in teaching order
}
```

**Logic**:
- Format context: abstractions + relationships
- Prompt LLM to determine pedagogical order
- Request YAML list of indices
- Validate:
  - All abstraction indices present exactly once
  - No duplicates
  - Indices within bounds
- Return ordered list

---

### 6. ChapterWriterService (Data Plane)

**Type**: `restate.Context` (Stateless Service)

**Handler**: `WriteChapter`

**Input**:
```go
type WriteChapterInput struct {
    Abstraction       Abstraction
    Files             []FileContent
    PreviousChapters  []ChapterSummary  // For context/continuity
    ProjectName       string
    ChapterNumber     int
}

type ChapterSummary struct {
    Name    string
    Summary string  // Brief summary for context
}
```

**Output**:
```go
type WriteChapterOutput struct {
    ChapterNumber int
    Title         string
    Content       string  // Markdown content
}
```

**Logic**:
- Format context:
  - Current abstraction details
  - Related file contents (by FileIndices)
  - Summaries of previous chapters (for coherence)
- Prompt LLM to write beginner-friendly chapter
- Request markdown format with:
  - Clear explanations
  - Code examples
  - Analogies
- Return chapter content

**Note**: Called in parallel for all chapters using `RequestFuture`

---

### 7. FileWriterService (Data Plane)

**Type**: `restate.Context` (Stateless Service)

**Handler**: `WriteMarkdownFiles`

**Input**:
```go
type WriteMarkdownFilesInput struct {
    OutputDir string
    Chapters  []WriteChapterOutput  // From ChapterWriterService
}
```

**Output**:
```go
type WriteMarkdownFilesOutput struct {
    FilesWritten []string  // Paths of created files
}
```

**Logic**:
- Create output directory if not exists
- For each chapter, write to file:
  - Filename: `{chapter_number:02d}_{sanitized_title}.md`
  - Example: `01_node_abstraction.md`
- Return list of written file paths

**Error Handling**:
- If write fails, return error (Saga will compensate)

---

## Type-Safe Client Definitions

```go
// Service clients for workflow to invoke
var (
    FileReaderClient = framework.ServiceClient[ReadFilesInput, ReadFilesOutput]{
        ServiceName: "FileReader",
        HandlerName: "ReadFiles",
    }

    AbstractionAnalyzerClient = framework.ServiceClient[AnalyzeAbstractionsInput, AnalyzeAbstractionsOutput]{
        ServiceName: "AbstractionAnalyzer",
        HandlerName: "AnalyzeAbstractions",
    }

    RelationshipAnalyzerClient = framework.ServiceClient[AnalyzeRelationshipsInput, RelationshipData]{
        ServiceName: "RelationshipAnalyzer",
        HandlerName: "AnalyzeRelationships",
    }

    ChapterOrdererClient = framework.ServiceClient[OrderChaptersInput, OrderChaptersOutput]{
        ServiceName: "ChapterOrderer",
        HandlerName: "OrderChapters",
    }

    ChapterWriterClient = framework.ServiceClient[WriteChapterInput, WriteChapterOutput]{
        ServiceName: "ChapterWriter",
        HandlerName: "WriteChapter",
    }

    FileWriterClient = framework.ServiceClient[WriteMarkdownFilesInput, WriteMarkdownFilesOutput]{
        ServiceName: "FileWriter",
        HandlerName: "WriteMarkdownFiles",
    }
)
```

## Project Structure

```
cb2utorial/
├── go.mod
├── go.sum
├── main.go                          # Server entry point
├── workflow/
│   └── tutorial_workflow.go         # TutorialWorkflow implementation
├── services/
│   ├── file_reader.go               # FileReaderService
│   ├── abstraction_analyzer.go      # AbstractionAnalyzerService
│   ├── relationship_analyzer.go     # RelationshipAnalyzerService
│   ├── chapter_orderer.go           # ChapterOrdererService
│   ├── chapter_writer.go            # ChapterWriterService
│   └── file_writer.go               # FileWriterService
├── types/
│   └── models.go                    # Shared type definitions
├── llm/
│   └── client.go                    # LLM client wrapper
├── utils/
│   ├── filewalker.go                # File traversal utilities
│   └── yaml.go                      # YAML parsing helpers
└── README.md
```

## Dependencies

```go
// go.mod
module github.com/yourusername/cb2utorial

go 1.23

require (
    github.com/pithomlabs/rea v0.x.x           // Rea framework
    github.com/restate-sdk-go v0.x.x           // Restate Go SDK
    github.com/gobwas/glob v0.2.3              // Pattern matching
    gopkg.in/yaml.v3 v3.0.1                    // YAML parsing
    github.com/sashabaranov/go-openai v1.x.x   // LLM client (or Gemini SDK)
)
```

## Implementation Steps

### Phase 1: Setup & Core Services
1. **Initialize Go project**:
   - `go mod init github.com/yourusername/cb2utorial`
   - Install dependencies: `go get github.com/pithomlabs/rea`

2. **Define shared types** (`types/models.go`):
   - All input/output structs
   - Ensure JSON serialization tags

3. **Implement FileReaderService** (`services/file_reader.go`):
   - File traversal logic
   - Pattern matching
   - Unit tests with mock filesystem

4. **Implement LLM client wrapper** (`llm/client.go`):
   - Abstract LLM provider (OpenAI/Gemini/etc.)
   - Retry logic
   - Environment variable configuration

### Phase 2: Analysis Services
5. **Implement AbstractionAnalyzerService** (`services/abstraction_analyzer.go`):
   - Prompt engineering
   - YAML response parsing
   - Validation logic

6. **Implement RelationshipAnalyzerService** (`services/relationship_analyzer.go`):
   - Similar to abstraction analyzer
   - Add relationship-specific validation

7. **Implement ChapterOrdererService** (`services/chapter_orderer.go`):
   - Ordering prompt
   - Validate completeness

### Phase 3: Writing Services
8. **Implement ChapterWriterService** (`services/chapter_writer.go`):
   - Context formatting with previous chapters
   - Markdown generation

9. **Implement FileWriterService** (`services/file_writer.go`):
   - File I/O
   - Filename sanitization

### Phase 4: Workflow Orchestration
10. **Implement TutorialWorkflow** (`workflow/tutorial_workflow.go`):
    - Sequential service calls
    - Saga setup with compensation
    - Parallel chapter writing using `RequestFuture`

11. **Setup server** (`main.go`):
    ```go
    func main() {
        server := restate.NewRestate()
        
        // Register services
        server.Bind(restate.Reflect(services.FileReaderService{}))
        server.Bind(restate.Reflect(services.AbstractionAnalyzerService{}))
        server.Bind(restate.Reflect(services.RelationshipAnalyzerService{}))
        server.Bind(restate.Reflect(services.ChapterOrdererService{}))
        server.Bind(restate.Reflect(services.ChapterWriterService{}))
        server.Bind(restate.Reflect(services.FileWriterService{}))
        
        // Register workflow
        server.Bind(restate.Reflect(workflow.TutorialWorkflow{}))
        
        server.Start(context.Background(), ":9080")
    }
    ```

### Phase 5: Testing & Integration
12. **Unit tests**: Test each service independently with mocked inputs
13. **Integration test**: Run full workflow against a sample repository
14. **CLI client**: Create a simple CLI to invoke the workflow

## CLI Client Example

```go
// cmd/generate/main.go
package main

import (
    "context"
    "flag"
    "log"
    
    "github.com/pithomlabs/rea"
    "github.com/yourusername/cb2utorial/types"
    "github.com/yourusername/cb2utorial/workflow"
)

func main() {
    repoPath := flag.String("repo", "", "Path to local repository")
    outputDir := flag.String("output", "./tutorial", "Output directory")
    flag.Parse()

    if *repoPath == "" {
        log.Fatal("--repo is required")
    }

    // Create workflow client
    client := framework.WorkflowClient[types.TutorialWorkflowInput, any]{
        WorkflowName: "TutorialWorkflow",
        HandlerName:  "Run",
    }

    input := types.TutorialWorkflowInput{
        LocalRepoPath: *repoPath,
        OutputDir:     *outputDir,
        MaxFiles:      100,
    }

    // Submit workflow (ingress call from external client)
    ctx := context.Background()
    workflowID, err := client.Submit(ctx, "tutorial-gen-1", input)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Workflow submitted: %s", workflowID)
    
    // Attach to workflow and wait for result
    result, err := client.Attach(ctx, workflowID)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Tutorial generated successfully: %+v", result)
}
```

## Key Rea Framework Features Used

### 1. Workflow Orchestration
- **WorkflowContext**: Durable execution with automatic journaling
- **State Persistence**: Store pipeline state across service calls
- **Automatic Retries**: Framework handles transient failures

### 2. Saga Pattern
- **Compensation Logic**: Clean up files on failure
- **LIFO Rollback**: Undo operations in reverse order
- **Idempotent Operations**: File deletion is naturally idempotent

### 3. Type-Safe Clients
- **ServiceClient[I, O]**: Compile-time type checking for service calls
- **WorkflowClient[I, O]**: Type-safe workflow invocation

### 4. Side Effect Management
- **Run Wrappers**: Wrap LLM calls for journaling and observability
- **Deterministic Replay**: Restate can replay workflows without re-calling LLMs

### 5. Parallel Execution
- **RequestFuture**: Generate multiple chapters concurrently
  ```go
  var futures []restate.ResponseFuture[WriteChapterOutput]
  for i, idx := range chapterOrder {
      future := ChapterWriterClient.RequestFuture(ctx, input)
      futures = append(futures, future)
  }
  // Collect results
  for _, future := range futures {
      result, err := future.Response()
      // handle result
  }
  ```

## Configuration

### Environment Variables
```bash
# LLM Configuration
export LLM_PROVIDER=openai          # or gemini, anthropic
export OPENAI_API_KEY=sk-...
export LLM_MODEL=gpt-4

# Restate Configuration
export RESTATE_SERVER_URL=http://localhost:9080

# File Processing
export MAX_FILE_SIZE=1048576        # 1MB default
export MAX_FILES=100
export INCLUDE_PATTERNS="*.go,*.py,*.js"
export EXCLUDE_PATTERNS="*_test.go,vendor/*,node_modules/*"
```

## Error Handling Strategy

1. **Service Level**:
   - Return typed errors
   - Log errors with context
   - Let Restate handle retries

2. **Workflow Level**:
   - Saga compensates on critical failures
   - Store error state in workflow
   - Provide detailed error messages to user

3. **LLM Failures**:
   - Automatic retry with exponential backoff
   - Fallback to simpler prompts if structured output fails
   - Log malformed LLM responses for debugging

## Testing Strategy

### Unit Tests
- **FileReaderService**: Test glob matching, size limits with mock filesystem
- **LLM Services**: Mock LLM client, test YAML parsing and validation
- **FileWriterService**: Test file creation and sanitization

### Integration Tests
1. **End-to-End Test**:
   - Use a small sample repository (e.g., 10 files)
   - Run full workflow
   - Verify markdown files are generated with correct structure
   - Clean up output directory

2. **Saga Test**:
   - Inject failure in middle of workflow
   - Verify compensation runs (files deleted)

### Manual Testing
1. Start Restate server: `restate-server`
2. Start service: `go run main.go`
3. Run CLI: `go run cmd/generate/main.go --repo /path/to/repo --output ./tutorial`
4. Inspect generated markdown files in `./tutorial/`

## Advantages Over Python PocketFlow

1. **Type Safety**: Compile-time validation prevents runtime errors
2. **Durability**: Restate provides automatic state persistence and recovery
3. **Observability**: Built-in tracing and logging via Restate
4. **Scalability**: Distributed execution for large codebases
5. **Performance**: Compiled binary with lower latency
6. **Error Recovery**: Automatic retries and saga compensation
7. **Concurrent Chapter Writing**: Native goroutines + Restate futures

## Trade-offs

1. **Complexity**: More setup than PocketFlow's simple in-memory execution
2. **Dependencies**: Requires Restate server running
3. **Learning Curve**: Understanding Restate concepts (workflows, journaling, etc.)
4. **Development Time**: More boilerplate due to type definitions

## Conclusion

This Go port using the rea framework transforms PocketFlow's simple pipeline into a production-grade, durable, distributed system. The control plane/data plane separation provides clear architecture, while Restate's durability and saga patterns ensure reliable execution even for long-running workflows.

The simplified version (local files + sequential markdown) reduces scope while demonstrating the core architectural patterns. Future enhancements could add:
- GitHub API integration
- Index.md with Mermaid diagrams
- Multi-language support
- Caching layer for LLM calls
- Web UI for workflow monitoring
