package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/pithomlabs/cb2utorial/llm"
	"github.com/pithomlabs/cb2utorial/types"
	restate "github.com/restatedev/sdk-go"
	"gopkg.in/yaml.v3"
)

// ChapterOrdererService determines pedagogical chapter order
type ChapterOrdererService struct{}

// ServiceName returns the service name for registration
func (s ChapterOrdererService) ServiceName() string {
	return "ChapterOrderer"
}

// OrderChapters uses LLM to determine best teaching sequence
func (s ChapterOrdererService) OrderChapters(ctx restate.Context, input types.OrderChaptersInput) (types.OrderChaptersOutput, error) {
	// Validate input
	if len(input.Abstractions) == 0 {
		return types.OrderChaptersOutput{}, fmt.Errorf("no abstractions provided")
	}

	// Build abstraction listing
	var abstractionListBuilder strings.Builder
	for _, abs := range input.Abstractions {
		abstractionListBuilder.WriteString(fmt.Sprintf("- %d # %s: %s\n", abs.Index, abs.Name, abs.Description))
	}

	// Build relationship context
	var relationshipBuilder strings.Builder
	relationshipBuilder.WriteString(fmt.Sprintf("Project Summary: %s\n\n", input.Relationships.Summary))
	relationshipBuilder.WriteString("Relationships:\n")
	for _, rel := range input.Relationships.Details {
		fromName := input.Abstractions[rel.FromIndex].Name
		toName := input.Abstractions[rel.ToIndex].Name
		relationshipBuilder.WriteString(fmt.Sprintf("- %d (%s) → %d (%s): %s\n",
			rel.FromIndex, fromName, rel.ToIndex, toName, rel.Label))
	}

	// Create LLM prompt
	prompt := fmt.Sprintf(`You are creating a tutorial for the "%s" project.

ABSTRACTIONS:
%s

CONTEXT:
%s

Your task: Determine the best order to explain these abstractions to a beginner.

Teaching strategy:
- Start with foundational concepts or user-facing entry points
- Progress to implementation details
- Ensure dependencies are explained before they're used
- Make it pedagogically sound

Return a YAML list of abstraction INDICES in teaching order.
Each entry should be: "index # Name" for clarity.

Example output:
"""yaml
- 2 # EntryPoint
- 0 # Foundation  
- 1 # Implementation
"""

IMPORTANT: Include ALL abstractions exactly once.
Return ONLY the YAML list, no other text.
`, input.ProjectName, abstractionListBuilder.String(), relationshipBuilder.String())

	// Call LLM
	client, err := llm.NewClient()
	if err != nil {
		return types.OrderChaptersOutput{}, fmt.Errorf("failed to create LLM client: %w", err)
	}

	response, err := client.CallLLM(context.Background(), prompt, "You are an expert technical educator.")
	if err != nil {
		return types.OrderChaptersOutput{}, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse YAML response
	var yamlIndices []interface{} // Can be int or "0 # Name"

	// Extract YAML block
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

	err = yaml.Unmarshal([]byte(yamlContent), &yamlIndices)
	if err != nil {
		return types.OrderChaptersOutput{}, fmt.Errorf("failed to parse YAML response: %w\nResponse: %s", err, response)
	}

	// Convert to integer indices
	orderedIndices := make([]int, len(yamlIndices))
	seen := make(map[int]bool)

	for i, val := range yamlIndices {
		idx, err := extractIndex(val)
		if err != nil {
			return types.OrderChaptersOutput{}, fmt.Errorf("failed to extract index at position %d: %w", i, err)
		}

		// Validate index
		if idx < 0 || idx >= len(input.Abstractions) {
			return types.OrderChaptersOutput{}, fmt.Errorf("index %d out of bounds at position %d", idx, i)
		}

		// Check for duplicates
		if seen[idx] {
			return types.OrderChaptersOutput{}, fmt.Errorf("duplicate index %d found", idx)
		}
		seen[idx] = true

		orderedIndices[i] = idx
	}

	// Verify all abstractions are included
	if len(orderedIndices) != len(input.Abstractions) {
		missing := []int{}
		for i := 0; i < len(input.Abstractions); i++ {
			if !seen[i] {
				missing = append(missing, i)
			}
		}
		return types.OrderChaptersOutput{}, fmt.Errorf("missing abstractions in order: %v", missing)
	}

	return types.OrderChaptersOutput{
		OrderedIndices: orderedIndices,
	}, nil
}
