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

// RelationshipAnalyzerService analyzes how abstractions interact
type RelationshipAnalyzerService struct{}

// ServiceName returns the service name for registration
func (s RelationshipAnalyzerService) ServiceName() string {
	return "RelationshipAnalyzer"
}

// AnalyzeRelationships generates project summary and relationship graph
func (s RelationshipAnalyzerService) AnalyzeRelationships(ctx restate.Context, input types.AnalyzeRelationshipsInput) (types.RelationshipData, error) {
	// Validate input
	if len(input.Abstractions) == 0 {
		return types.RelationshipData{}, fmt.Errorf("no abstractions provided")
	}

	// Build abstraction listing
	var abstractionListBuilder strings.Builder
	for _, abs := range input.Abstractions {
		abstractionListBuilder.WriteString(fmt.Sprintf("- %d # %s: %s\n", abs.Index, abs.Name, abs.Description))
	}

	// Build code context for each abstraction (sample only)
	var codeContextBuilder strings.Builder
	for _, abs := range input.Abstractions {
		codeContextBuilder.WriteString(fmt.Sprintf("\n### Abstraction %d: %s\n", abs.Index, abs.Name))
		codeContextBuilder.WriteString("Related files:\n")
		for _, fileIdx := range abs.FileIndices {
			if fileIdx < len(input.Files) {
				file := input.Files[fileIdx]
				// Show first 500 chars as sample
				sample := file.Content
				if len(sample) > 500 {
					sample = sample[:500] + "..."
				}
				codeContextBuilder.WriteString(fmt.Sprintf("  File %d (%s):\n%s\n\n", fileIdx, file.Path, sample))
			}
		}
	}

	// Create LLM prompt
	prompt := fmt.Sprintf(`You are analyzing relationships in the "%s" project.

ABSTRACTIONS:
%s

CODE CONTEXT:
%s

Your tasks:
1. Write a high-level project summary (2-3 sentences)
2. Describe how these abstractions relate to each other

For relationships, specify:
- from: Source abstraction INDEX (number)
- to: Target abstraction INDEX (number)  
- label: Brief description of relationship (e.g., "uses", "extends", "orchestrates")

Output YAML format:
"""yaml
summary: "High-level description of what this project does"
details:
  - from: 0
    to: 1
    label: "uses"
  - from: 2
    to: 0
    label: "orchestrates"
"""

Return ONLY the YAML, no other text.
`, input.ProjectName, abstractionListBuilder.String(), codeContextBuilder.String())

	// Call LLM
	client, err := llm.NewClient()
	if err != nil {
		return types.RelationshipData{}, fmt.Errorf("failed to create LLM client: %w", err)
	}

	response, err := client.CallLLM(context.Background(), prompt, "You are a software architecture analyst.")
	if err != nil {
		return types.RelationshipData{}, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse YAML response
	type yamlRelationship struct {
		From  interface{} `yaml:"from"` // Can be int or "0 # Name"
		To    interface{} `yaml:"to"`
		Label string      `yaml:"label"`
	}

	type yamlRelationshipData struct {
		Summary string             `yaml:"summary"`
		Details []yamlRelationship `yaml:"details"`
	}

	var yamlData yamlRelationshipData

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

	err = yaml.Unmarshal([]byte(yamlContent), &yamlData)
	if err != nil {
		return types.RelationshipData{}, fmt.Errorf("failed to parse YAML response: %w\nResponse: %s", err, response)
	}

	// Convert to output format
	relationships := make([]types.Relationship, len(yamlData.Details))
	for i, yr := range yamlData.Details {
		// Extract indices (handle both int and "0 # Name" formats)
		fromIdx, err := extractIndex(yr.From)
		if err != nil {
			return types.RelationshipData{}, fmt.Errorf("invalid 'from' index in relationship %d: %w", i, err)
		}

		toIdx, err := extractIndex(yr.To)
		if err != nil {
			return types.RelationshipData{}, fmt.Errorf("invalid 'to' index in relationship %d: %w", i, err)
		}

		// Validate indices
		if fromIdx < 0 || fromIdx >= len(input.Abstractions) {
			return types.RelationshipData{}, fmt.Errorf("from index %d out of bounds in relationship %d", fromIdx, i)
		}
		if toIdx < 0 || toIdx >= len(input.Abstractions) {
			return types.RelationshipData{}, fmt.Errorf("to index %d out of bounds in relationship %d", toIdx, i)
		}

		relationships[i] = types.Relationship{
			FromIndex: fromIdx,
			ToIndex:   toIdx,
			Label:     yr.Label,
		}
	}

	return types.RelationshipData{
		Summary: yamlData.Summary,
		Details: relationships,
	}, nil
}
