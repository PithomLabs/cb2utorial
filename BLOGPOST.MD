# Building an Unbreakable AI Agent: Porting Code2Tutorial to Go with Restate

I recently took on the challenge of porting the [PocketFlow Tutorial Generator](https://github.com/The-Pocket/PocketFlow-Tutorial-Codebase-Knowledge) from Python to Go. The goal? To build `cb2utorial`—an AI agent that reads a codebase and writes a comprehensive, chapter-by-chapter tutorial explaining how it works.

This isn't just a simple script; it's a complex workflow involving file parsing, deep code analysis, relationship mapping, and generating multiple chapters of content using LLMs. In a traditional setup, this is fragile. If the LLM times out on Chapter 4, you lose everything and start over.

Enter **Restate** and the **Rea** framework.


Building this with the [Restate Go SDK](https://docs.restate.dev/develop/go) felt like cheating. Restate provides "durable execution," which means my code is guaranteed to run to completion.

If a network glitch happens during an API call? Restate retries it.
If the server crashes while writing Chapter 3? Restate brings it back up and resumes *exactly* where it left off.

I didn't have to write a single line of state recovery code.

Why Durable Execution Changes the Game

Here is why Restate was the perfect backbone for this AI agent:

LLMs are Slow and Flaky

Generating a full tutorial takes minutes. In the world of HTTP, that's an eternity. With Restate, I didn't worry about timeouts. I could fire off an async workflow, let it churn for 10 minutes, and poll for the result. The state is persisted automatically.

"Monolithic" Simplicity for Distributed Logic

Using the **Rea** framework (a Go orchestration layer for Restate), I defined my microservices—`FileReader`, `AbstractionAnalyzer`, `ChapterWriter`—but orchestrated them as if they were just local function calls.

```go
// It looks like local code, but it's a distributed workflow!
files := FileReader.Call(ctx, input)
abstractions := AbstractionAnalyzer.Call(ctx, files)
chapters := ChapterWriter.Call(ctx, abstractions)
```



Porting `cb2utorial` to Go wasn't just about changing languages; it was about upgrading the architecture. By building on Restate, I moved from a "script" that runs on my machine to a **resilient, distributed application** that can scale.

For anyone building long-running AI workflows: stop building fragile scripts. Embrace durable execution. It’s the infrastructure AI agents deserve.
