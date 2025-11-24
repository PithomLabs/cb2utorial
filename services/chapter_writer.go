package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/pithomlabs/cb2utorial/llm"
	"github.com/pithomlabs/cb2utorial/types"
	restate "github.com/restatedev/sdk-go"
)

// ChapterWriterService generates markdown tutorial chapters
type ChapterWriterService struct{}

// ServiceName returns the service name for registration
func (s ChapterWriterService) ServiceName() string {
	return "ChapterWriter"
}

// WriteChapter creates a detailed tutorial chapter for one abstraction
func (s ChapterWriterService) WriteChapter(ctx restate.Context, input types.WriteChapterInput) (types.WriteChapterOutput, error) {
	// Validate input
	if input.Abstraction.Name == "" {
		return types.WriteChapterOutput{}, fmt.Errorf("abstraction name is required")
	}

	// Build context of related files
	var fileContextBuilder strings.Builder
	fileContextBuilder.WriteString("Related code files:\n\n")

	for _, fileIdx := range input.Abstraction.FileIndices {
		if fileIdx >= len(input.Files) {
			continue // Skip invalid indices
		}

		file := input.Files[fileIdx]
		fileContextBuilder.WriteString(fmt.Sprintf("### File: %s\n", file.Path))
		fileContextBuilder.WriteString("```\n")

		// Truncate very long files
		content := file.Content
		const maxContentLength = 8000
		if len(content) > maxContentLength {
			content = content[:maxContentLength] + "\n... (truncated for brevity)"
		}

		fileContextBuilder.WriteString(content)
		fileContextBuilder.WriteString("\n```\n\n")
	}

	// Build context of previous chapters
	var previousChaptersContext string
	if len(input.PreviousChapters) > 0 {
		var prevBuilder strings.Builder
		prevBuilder.WriteString("\n\nPREVIOUSLY COVERED CONCEPTS (for reference, don't repeat):\n")
		for _, prev := range input.PreviousChapters {
			prevBuilder.WriteString(fmt.Sprintf("- %s: %s\n", prev.Name, prev.Summary))
		}
		previousChaptersContext = prevBuilder.String()
	}

	// Create LLM prompt
	prompt := fmt.Sprintf(`You are writing a tutorial chapter for the "%s" project.

TARGET AUDIENCE: Developers new to this codebase who want to understand it quickly.

ABSTRACTION TO EXPLAIN:
Name: %s
Description: %s

%s

%s

Your task: Write a comprehensive, beginner-friendly tutorial chapter explaining this abstraction.

REQUIREMENTS:
1. Use clear, simple language
2. Include code examples from the provided files
3. Use analogies or real-world examples where helpful
4. Explain WHY this abstraction exists, not just WHAT it does
5. Break down complex concepts into digestible parts
6. Format as markdown

STRUCTURE YOUR CHAPTER:
# %s

[Brief introduction - what is this and why does it matter?]

## What It Does

[Clear explanation of the abstraction's purpose]

## Key Code

[Show relevant code snippets with explanations]

## How It Works

[Step-by-step explanation of the implementation]

## Key Takeaways

- [Important point 1]
- [Important point 2]
- [Important point 3]

OUTPUT: Return ONLY the markdown content, no meta-commentary.
`,
		input.ProjectName,
		input.Abstraction.Name,
		input.Abstraction.Description,
		fileContextBuilder.String(),
		previousChaptersContext,
		input.Abstraction.Name,
	)

	// Call LLM
	client, err := llm.NewClient()
	if err != nil {
		return types.WriteChapterOutput{}, fmt.Errorf("failed to create LLM client: %w", err)
	}

	systemPrompt := "You are an expert technical educator who excels at explaining complex code in simple terms."

	response, err := client.CallLLM(context.Background(), prompt, systemPrompt)
	if err != nil {
		return types.WriteChapterOutput{}, fmt.Errorf("LLM call failed: %w", err)
	}

	// Clean up response (remove any markdown code fences if LLM wrapped the output)
	content := strings.TrimSpace(response)
	if strings.HasPrefix(content, "```markdown") {
		content = strings.TrimPrefix(content, "```markdown")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	return types.WriteChapterOutput{
		ChapterNumber: input.ChapterNumber,
		Title:         input.Abstraction.Name,
		Content:       content,
	}, nil
}
