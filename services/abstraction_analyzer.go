package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pithomlabs/cb2utorial/llm"
	"github.com/pithomlabs/cb2utorial/types"
	restate "github.com/restatedev/sdk-go"
	"gopkg.in/yaml.v3"
)

// AbstractionAnalyzerService identifies core abstractions from code
type AbstractionAnalyzerService struct{}

// ServiceName returns the service name for registration
func (s AbstractionAnalyzerService) ServiceName() string {
	return "AbstractionAnalyzer"
}

// AnalyzeAbstractions uses LLM to identify key code concepts
func (s AbstractionAnalyzerService) AnalyzeAbstractions(ctx restate.Context, input types.AnalyzeAbstractionsInput) (types.AnalyzeAbstractionsOutput, error) {
	// Validate input
	if len(input.Files) == 0 {
		return types.AnalyzeAbstractionsOutput{}, fmt.Errorf("no files provided")
	}

	// Build file context with indices
	var contextBuilder strings.Builder
	for _, file := range input.Files {
		contextBuilder.WriteString(fmt.Sprintf("--- File Index %d: %s ---\n", file.Index, file.Path))
		// Truncate very long files for context
		content := file.Content
		if len(content) > 5000 {
			content = content[:5000] + "\n... (truncated)"
		}
		contextBuilder.WriteString(content)
		contextBuilder.WriteString("\n\n")
	}

	// Build file listing for reference
	var fileListBuilder strings.Builder
	for _, file := range input.Files {
		fileListBuilder.WriteString(fmt.Sprintf("- %d # %s\n", file.Index, file.Path))
	}

	// Create LLM prompt
	prompt := fmt.Sprintf(`You are analyzing the codebase for project "%s".

FILES:
%s

FILE LISTING (for reference):
%s

Your task: Identify the 5-10 core abstractions/concepts in this codebase.

For each abstraction, provide:
- name: A clear, concise name
- description: Beginner-friendly explanation (1-2 sentences)
- files: List of file INDICES (numbers only) related to this abstraction

Output YAML format:
"""yaml
- name: "CoreAbstraction"
  description: "What this abstraction represents and why it matters"
  files: [0, 3, 5]
- name: "AnotherConcept"
  description: "Another key concept"
  files: [1, 2]
"""

Focus on the most important abstractions that a newcomer should understand.
Return ONLY the YAML, no other text.
`, input.ProjectName, contextBuilder.String(), fileListBuilder.String())

	// Call LLM
	client, err := llm.NewClient()
	if err != nil {
		return types.AnalyzeAbstractionsOutput{}, fmt.Errorf("failed to create LLM client: %w", err)
	}

	response, err := client.CallLLM(context.Background(), prompt, "You are a code analysis expert helping developers understand unfamiliar codebases.")
	if err != nil {
		return types.AnalyzeAbstractionsOutput{}, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse YAML response
	type yamlAbstraction struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Files       []int  `yaml:"files"`
	}

	var yamlAbstractions []yamlAbstraction

	// Extract YAML block if wrapped in code fence
	yamlContent := response
	if strings.Contains(response, "```yaml") {
		parts := strings.Split(response, "```yaml")
		if len(parts) > 1 {
			yamlContent = strings.Split(parts[1], "```")[0]
		}
	} else if strings.Contains(response, "```") {
		parts := strings.Split(response, "```")
		if len(parts) > 1 {
			yamlContent = parts[1]
		}
	}

	err = yaml.Unmarshal([]byte(yamlContent), &yamlAbstractions)
	if err != nil {
		return types.AnalyzeAbstractionsOutput{}, fmt.Errorf("failed to parse YAML response: %w\nResponse: %s", err, response)
	}

	// Validate and convert to output format
	if len(yamlAbstractions) == 0 {
		return types.AnalyzeAbstractionsOutput{}, fmt.Errorf("no abstractions identified")
	}

	if len(yamlAbstractions) > input.MaxAbstractions {
		yamlAbstractions = yamlAbstractions[:input.MaxAbstractions]
	}

	abstractions := make([]types.Abstraction, len(yamlAbstractions))
	for i, ya := range yamlAbstractions {
		// Validate file indices
		for _, fileIdx := range ya.Files {
			if fileIdx < 0 || fileIdx >= len(input.Files) {
				return types.AnalyzeAbstractionsOutput{}, fmt.Errorf("invalid file index %d in abstraction %s", fileIdx, ya.Name)
			}
		}

		abstractions[i] = types.Abstraction{
			Index:       i,
			Name:        ya.Name,
			Description: ya.Description,
			FileIndices: ya.Files,
		}
	}

	return types.AnalyzeAbstractionsOutput{
		Abstractions: abstractions,
	}, nil
}

// extractIndex handles both int and "0 # Name" formats
func extractIndex(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case string:
		// Handle "0 # Name" format
		if strings.Contains(v, "#") {
			parts := strings.Split(v, "#")
			return strconv.Atoi(strings.TrimSpace(parts[0]))
		}
		return strconv.Atoi(strings.TrimSpace(v))
	default:
		return 0, fmt.Errorf("unexpected type %T for index", value)
	}
}
