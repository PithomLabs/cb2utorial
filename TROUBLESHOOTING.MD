# cb2utorial Development Documentation

This document chronicles the complete debugging and development process for the cb2utorial (Codebase to Tutorial) application, including all bugs encountered and their solutions.

## Project Overview

**Goal**: Port Python-based tutorial generator to Go using:
- `github.com/restatedev/sdk-go` (v0.22.0) for durable execution
- `github.com/pithomlabs/rea` (v0.1.0) as orchestration framework
- `github.com/revrost/go-openrouter` for LLM integration

**Architecture**: 
- Microservices: FileReader, AbstractionAnalyzer, RelationshipAnalyzer, ChapterOrderer, ChapterWriter, FileWriter
- Orchestration: TutorialWorkflow (Restate workflow)
- Deployment: Service on port 9082, Restate server on ports 8080 (ingress) and 9070 (admin)

---

## Bug #1: BIDI_STREAM Service Binding Error

### Problem
When starting the application, the following error occurred:
```
Failed to bind FileReader: &{...map[FileReader:0xc00019e750] ... BIDI_STREAM}
exit status 1
```

### Initial Hypothesis (INCORRECT)
Service name mismatch between struct name (`FileReaderService`) and expected name (`FileReader`).

### First Attempted Fix
Added `ServiceName()` methods to all services:
```go
func (s FileReaderService) ServiceName() string {
    return "FileReader"
}
```

**Result**: Error persisted, but service name in error changed to `FileReader`, confirming the name was now correct.

### Root Cause Analysis
Through debugging:
1. Created minimal test service - same error occurred
2. Printed error type: `*server.Restate` instead of actual error
3. Checked SDK documentation: `Bind()` returns `*Restate`, NOT an error!

**Actual Root Cause**: The `server.Bind()` method uses **builder pattern** and returns `*Restate` for method chaining, not an error.

### Solution
Changed from error-checking pattern to method chaining:

```go
// BEFORE (WRONG)
if err := server.Bind(restate.Reflect(services.FileReaderService{})); err != nil {
    log.Fatalf("Failed to bind: %v", err)
}

// AFTER (CORRECT)
server := server.NewRestate().
    Bind(restate.Reflect(services.FileReaderService{})).
    Bind(restate.Reflect(services.AbstractionAnalyzerService{})).
    Bind(restate.Reflect(services.RelationshipAnalyzerService{})).
    Bind(restate.Reflect(services.ChapterOrdererService{})).
    Bind(restate.Reflect(services.ChapterWriterService{})).
    Bind(restate.Reflect(services.FileWriterService{})).
    Bind(restate.Reflect(workflow.TutorialWorkflow{}))
```

**Files Modified**: `main.go`

---

## Bug #2: Workflow Invocation 404 Error

### Problem
After fixing service binding, the CLI failed with:
```
POST http://localhost:9070/TutorialWorkflow/Run
Workflow failed with status 404
```

### Investigation Process

#### Attempt 1: Missing Workflow ID
Hypothesis: Workflows require an ID in the URL path.

Tested: `POST /TutorialWorkflow/workflow-123/Run`
Result: Still 404

#### Attempt 2: Handler Name Case
Hypothesis: Handler name should be lowercase `run` (from TypeScript docs).

Tested: `POST /TutorialWorkflow/workflow-123/run`
Result: Still 404

Checked Restate playground - shows **uppercase** `Run` (Go method name is preserved)

#### Attempt 3: Service Registration
Hypothesis: Service not registered with Restate server.

Action:
```bash
curl -X POST http://localhost:9070/deployments \
  -H "Content-Type: application/json" \
  -d '{"uri": "http://localhost:9082"}'
```

Result: Service registered successfully, but still 404

#### Attempt 4: Port Investigation (SUCCESS!)
Checked listening ports:
```bash
netstat -tlnp | grep restate
tcp  0.0.0.0:9070  (restate-server - Admin API)
tcp  0.0.0.0:8080  (restate-server - Ingress)
tcp  :::9082        (cb2utorial service)
```

**Discovery**: Restate uses TWO ports:
- **Port 9070**: Admin API (service registration, discovery)
- **Port 8080**: Ingress (actual workflow/service invocations)

### Root Cause
Using wrong port (9070 instead of 8080) for workflow invocations.

### Solution
Changed CLI default URL from port 9070 to 8080:

```go
// cmd/cli/main.go
restateURL := flag.String("restate-url", "http://localhost:8080", "Restate server ingress URL")
```

Correct endpoint format:
```
POST http://localhost:8080/TutorialWorkflow/{workflowId}/Run
```

**Files Modified**: `cmd/cli/main.go`

---

## Bug #3: Silent Execution (No Progress Feedback)

### Problem
Workflow invoked successfully but CLI appeared stuck with no output for 5-15 minutes.

### Root Cause
LLM API calls are slow (10-60 seconds each), and the workflow makes multiple sequential calls:
1. AbstractionAnalyzer (1 call)
2. RelationshipAnalyzer (1 call)  
3. ChapterOrderer (1 call)
4. ChapterWriter (N calls, one per chapter)

No logging means users can't see progress.

### Solution
Added comprehensive logging to workflow steps:

```go
fmt.Printf("🚀 Starting TutorialWorkflow for repo: %s\n", input.LocalRepoPath)
fmt.Printf("📁 Step 1/6: Reading files from %s...\n", input.LocalRepoPath)
fmt.Printf("✅ Found %d files\n", len(filesOutput.Files))
fmt.Printf("🔍 Step 2/6: Analyzing code abstractions (calling LLM)...\n")
fmt.Printf("✅ Identified %d abstractions\n", len(abstractionsOutput.Abstractions))
fmt.Printf("🔗 Step 3/6: Analyzing relationships (calling LLM)...\n")
fmt.Printf("✅ Mapped relationships\n")
fmt.Printf("📋 Step 4/6: Ordering chapters (calling LLM)...\n")
fmt.Printf("✅ Chapter order determined\n")
fmt.Printf("✍️  Step 5/6: Generating %d chapters (calling LLM for each)...\n", len(orderOutput.OrderedIndices))
fmt.Printf("  📝 Writing chapter %d/%d: %s...\n", i+1, len(orderOutput.OrderedIndices), abstraction.Name)
fmt.Printf("💾 Step 6/6: Writing markdown files...\n")
fmt.Printf("🎉 Tutorial generation complete! %d files written.\n", len(result.FilesWritten))
```

**Files Modified**: `workflow/tutorial_workflow.go`

---

## Key Architecture Insights

### Restate Port Configuration

| Port | Purpose | Usage |
|------|---------|-------|
| 8080 | Ingress API | Invoke workflows and services |
| 9070 | Admin API | Register deployments, query service metadata |
| 9082 | Service | Your cb2utorial service endpoint |

### Restate Workflow Invocation Pattern

```
┌─────────────┐         ┌──────────────────┐         ┌──────────────────┐
│   Client    │────────▶│  Restate Server  │────────▶│  Your Service    │
│   (CLI)     │  :8080  │   (Ingress)      │  :9082  │  (cb2utorial)    │
└─────────────┘         │                  │         └──────────────────┘
                        │  - Journaling    │
                        │  - State Store   │
                        │  - Retry Logic   │
                        └──────────────────┘
```

**Critical**: Always invoke through Restate ingress (port 8080), never directly to service (port 9082).

### Service Binding with Restate SDK

The `Bind()` method follows the **builder pattern**:
```go
func (r *Restate) Bind(definition restate.ServiceDefinition) *Restate
```

Returns the server itself for chaining, NOT an error. This is different from most Go APIs and can be confusing.

### Workflow Endpoint Format

For Go services with method `Run`:
```
POST http://localhost:8080/{WorkflowName}/{workflowId}/Run
```

- Handler name preserves Go method capitalization
- Workflow ID is required in the path
- TypeScript examples use lowercase `run`, but Go uses actual method name

---

## Setup Process (Final Working Configuration)

### 1. Start Restate Server
```bash
restate-server
# Listens on ports 8080 (ingress) and 9070 (admin)
```

### 2. Build and Start cb2utorial Service
```bash
go build -o cb2utorial main.go
./cb2utorial
# Listens on port 9082
```

### 3. Register Service with Restate (One-time)
```bash
curl -X POST http://localhost:9070/deployments \
  -H "Content-Type: application/json" \
  -d '{"uri": "http://localhost:9082"}'
```

### 4. Set Environment Variables
```bash
export OPENROUTER_API_KEY="your_api_key"
```

### 5. Run CLI
```bash
cd cmd/cli
./cli --repo /path/to/repo --output ./tutorial
```

---

## Development Timeline

1. **Initial Build**: All services compiled successfully
2. **Service Binding Issue**: Discovered `Bind()` builder pattern (1 hour debugging)
3. **404 Error**: Traced through endpoint format and service registration (2 hours)
4. **Port Discovery**: Found 8080 vs 9070 distinction (30 minutes)
5. **Progress Logging**: Added workflow step logging (30 minutes)
6. **First Successful Run**: Generated 5-chapter tutorial from rea-master codebase

**Total Debug Time**: ~4 hours
**Final Result**: Fully functional tutorial generator

---

## Lessons Learned

### 1. Read SDK Documentation Carefully
The Restate SDK uses builder patterns extensively. Don't assume Go conventions.

### 2. Understand Multi-Port Services
Restate's separation of admin (9070) and ingress (8080) is crucial for proper operation.

### 3. Add Logging Early
LLM operations are slow. Without progress feedback, users assume the app is broken.

### 4. Test Incrementally
Creating minimal test cases (like `test_minimal.go`) helped isolate the builder pattern issue.

### 5. Use cURL for Debugging
Testing endpoints directly with cURL isolated client vs server issues.

### 6. Check Restate Playground
The playground shows exact endpoint formats and expected request bodies for your specific services.

---

## Future Improvements

### Async Invocation
Currently the CLI blocks until workflow completes. Could use `/send` endpoint:
```go
// Fire and forget
POST http://localhost:8080/TutorialWorkflow/{id}/Run/send

// Returns immediately with invocation ID
// Poll status later: GET http://localhost:9070/invocations/{id}
```

### Parallel Chapter Generation
Currently chapters are written sequentially. Could use `restate.WaitFirst()` or `restate.Wait()` for parallel LLM calls.

### Error Recovery
Add retry logic and better error messages for LLM failures.

### Configuration File
Support `.cb2utorialrc` for default settings (max files, output dir, etc.).

---

## Verification

Successful run generates:
```
./tutorial/
  ├── 01_service_types.md
  ├── 02_framework_policy_control.md
  ├── 03_void_type.md
  ├── 04_idempotency_validation.md
  └── 05_guardrail_violations.md
```

Each file contains:
- Introduction with analogy
- What it does
- Key code snippets
- How it works
- Key takeaways

**End Result**: High-quality, pedagogically-ordered tutorial content generated automatically from source code analysis.

---

## References

- [Restate Go SDK](https://docs.restate.dev/develop/go)
- [Restate HTTP Invocation](https://docs.restate.dev/services/invocation/http)
- [rea Framework](https://github.com/pithomlabs/rea)
- [OpenRouter API](https://openrouter.ai/docs)

**Last Updated**: November 23, 2025
