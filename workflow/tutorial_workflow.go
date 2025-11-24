package workflow

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pithomlabs/cb2utorial/types"
	framework "github.com/pithomlabs/rea"
	restate "github.com/restatedev/sdk-go"
)

// Service clients using rea framework
var (
	FileReaderClient = framework.ServiceClient[types.ReadFilesInput, types.ReadFilesOutput]{
		ServiceName: "FileReader",
		HandlerName: "ReadFiles",
	}

	AbstractionAnalyzerClient = framework.ServiceClient[types.AnalyzeAbstractionsInput, types.AnalyzeAbstractionsOutput]{
		ServiceName: "AbstractionAnalyzer",
		HandlerName: "AnalyzeAbstractions",
	}

	RelationshipAnalyzerClient = framework.ServiceClient[types.AnalyzeRelationshipsInput, types.RelationshipData]{
		ServiceName: "RelationshipAnalyzer",
		HandlerName: "AnalyzeRelationships",
	}

	ChapterOrdererClient = framework.ServiceClient[types.OrderChaptersInput, types.OrderChaptersOutput]{
		ServiceName: "ChapterOrderer",
		HandlerName: "OrderChapters",
	}

	ChapterWriterClient = framework.ServiceClient[types.WriteChapterInput, types.WriteChapterOutput]{
		ServiceName: "ChapterWriter",
		HandlerName: "WriteChapter",
	}

	FileWriterClient = framework.ServiceClient[types.WriteMarkdownFilesInput, types.WriteMarkdownFilesOutput]{
		ServiceName: "FileWriter",
		HandlerName: "WriteMarkdownFiles",
	}
)

// TutorialWorkflow orchestrates the entire tutorial generation pipeline
type TutorialWorkflow struct{}

// ServiceName returns the service name for registration
func (w TutorialWorkflow) ServiceName() string {
	return "TutorialWorkflow"
}

// Run executes the complete workflow using rea framework service clients
func (w TutorialWorkflow) Run(ctx restate.WorkflowContext, input types.TutorialWorkflowInput) (types.WriteMarkdownFilesOutput, error) {
	// Log workflow start
	fmt.Printf("🚀 Starting TutorialWorkflow for repo: %s\n", input.LocalRepoPath)

	// Derive project name from path if not provided
	projectName := input.ProjectName
	if projectName == "" {
		projectName = filepath.Base(input.LocalRepoPath)
		if projectName == "." || projectName == "/" {
			projectName = "Project"
		}
	}

	// Get configuration from environment or use defaults
	maxFiles := input.MaxFiles
	if maxFiles == 0 {
		maxFiles = 100
	}

	// Step 1: Read Files
	fmt.Printf("📁 Step 1/6: Reading files from %s...\n", input.LocalRepoPath)
	fileReaderInput := types.ReadFilesInput{
		RepoPath:        input.LocalRepoPath,
		IncludePatterns: []string{"*.go", "*.py", "*.js", "*.ts", "*.java", "*.rb", "*.md"},
		ExcludePatterns: []string{"*_test.go", "vendor/*", "node_modules/*", ".git/*", "*.min.js"},
		MaxFileSize:     1048576, // 1MB
		MaxFiles:        maxFiles,
	}

	filesOutput, err := FileReaderClient.Call(ctx, fileReaderInput)
	if err != nil {
		return types.WriteMarkdownFilesOutput{}, fmt.Errorf("failed to read files: %w", err)
	}

	if len(filesOutput.Files) == 0 {
		return types.WriteMarkdownFilesOutput{}, fmt.Errorf("no files found in repository")
	}
	fmt.Printf("✅ Found %d files\n", len(filesOutput.Files))

	// Step 2: Identify Abstractions
	fmt.Printf("🔍 Step 2/6: Analyzing code abstractions (calling LLM)...\n")
	abstractionInput := types.AnalyzeAbstractionsInput{
		Files:           filesOutput.Files,
		ProjectName:     projectName,
		MaxAbstractions: 10,
	}

	abstractionsOutput, err := AbstractionAnalyzerClient.Call(ctx, abstractionInput)
	if err != nil {
		return types.WriteMarkdownFilesOutput{}, fmt.Errorf("failed to analyze abstractions: %w", err)
	}

	if len(abstractionsOutput.Abstractions) == 0 {
		return types.WriteMarkdownFilesOutput{}, fmt.Errorf("no abstractions identified")
	}
	fmt.Printf("✅ Identified %d abstractions\n", len(abstractionsOutput.Abstractions))

	// Step 3: Analyze Relationships
	fmt.Printf("🔗 Step 3/6: Analyzing relationships (calling LLM)...\n")
	relationshipInput := types.AnalyzeRelationshipsInput{
		Abstractions: abstractionsOutput.Abstractions,
		Files:        filesOutput.Files,
		ProjectName:  projectName,
	}

	relationships, err := RelationshipAnalyzerClient.Call(ctx, relationshipInput)
	if err != nil {
		return types.WriteMarkdownFilesOutput{}, fmt.Errorf("failed to analyze relationships: %w", err)
	}
	fmt.Printf("✅ Mapped relationships\n")

	// Step 4: Order Chapters
	fmt.Printf("📋 Step 4/6: Ordering chapters (calling LLM)...\n")
	orderInput := types.OrderChaptersInput{
		Abstractions:  abstractionsOutput.Abstractions,
		Relationships: relationships,
		ProjectName:   projectName,
	}

	orderOutput, err := ChapterOrdererClient.Call(ctx, orderInput)
	if err != nil {
		return types.WriteMarkdownFilesOutput{}, fmt.Errorf("failed to order chapters: %w", err)
	}
	fmt.Printf("✅ Chapter order determined\n")

	// Step 5: Write Chapters (sequentially - parallel can use RequestFuture later)
	fmt.Printf("✍️  Step 5/6: Generating %d chapters (calling LLM for each)...\n", len(orderOutput.OrderedIndices))
	chapters := make([]types.WriteChapterOutput, len(orderOutput.OrderedIndices))
	previousChapters := []types.ChapterSummary{}

	for i, absIndex := range orderOutput.OrderedIndices {
		abstraction := abstractionsOutput.Abstractions[absIndex]

		fmt.Printf("  📝 Writing chapter %d/%d: %s...\n", i+1, len(orderOutput.OrderedIndices), abstraction.Name)
		chapterInput := types.WriteChapterInput{
			Abstraction:      abstraction,
			Files:            filesOutput.Files,
			PreviousChapters: previousChapters,
			ProjectName:      projectName,
			ChapterNumber:    i + 1,
		}

		chapterOutput, err := ChapterWriterClient.Call(ctx, chapterInput)
		if err != nil {
			return types.WriteMarkdownFilesOutput{}, fmt.Errorf("failed to write chapter %d: %w", i+1, err)
		}

		chapters[i] = chapterOutput

		// Add to previous chapters context (summary = first 200 chars)
		summary := chapterOutput.Content
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		// Remove markdown headers and extra whitespace
		summary = strings.ReplaceAll(summary, "#", "")
		summary = strings.TrimSpace(summary)

		previousChapters = append(previousChapters, types.ChapterSummary{
			Name:    chapterOutput.Title,
			Summary: summary,
		})
	}

	// Step 6: Write Files
	fmt.Printf("💾 Step 6/6: Writing markdown files...\n")
	writerInput := types.WriteMarkdownFilesInput{
		OutputDir: input.OutputDir,
		Chapters:  chapters,
	}

	result, err := FileWriterClient.Call(ctx, writerInput)
	if err != nil {
		return types.WriteMarkdownFilesOutput{}, fmt.Errorf("failed to write markdown files: %w", err)
	}
	fmt.Printf("🎉 Tutorial generation complete! %d files written.\n", len(result.FilesWritten))

	return result, nil
}
