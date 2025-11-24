package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/pithomlabs/cb2utorial/types"
)

func main() {
	// Parse command-line flags
	repoPath := flag.String("repo", "", "Path to local repository (required)")
	outputDir := flag.String("output", "./tutorial", "Output directory for tutorial files")
	projectName := flag.String("project", "", "Project name (optional, derived from repo if empty)")
	maxFiles := flag.Int("max-files", 100, "Maximum number of files to process")
	restateURL := flag.String("restate-url", "http://localhost:8080", "Restate server ingress URL")

	flag.Parse()

	// Validate required flags
	if *repoPath == "" {
		log.Fatal("--repo flag is required")
	}

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Validate environment
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}

	// Create workflow input
	input := types.TutorialWorkflowInput{
		LocalRepoPath: *repoPath,
		OutputDir:     *outputDir,
		MaxFiles:      *maxFiles,
		ProjectName:   *projectName,
	}

	log.Printf("Generating tutorial for: %s", *repoPath)
	log.Printf("Output directory: %s", *outputDir)
	log.Printf("Max files: %d", *maxFiles)

	// Generate workflow ID from repo path and timestamp
	workflowID := fmt.Sprintf("tutorial-%d", time.Now().Unix())

	// Invoke workflow via Restate HTTP ingress
	// Endpoint format: POST /{WorkflowName}/{workflowId}/Run (matches Go method name)
	url := fmt.Sprintf("%s/TutorialWorkflow/%s/Run", *restateURL, workflowID)

	// Serialize input
	inputJSON, err := json.Marshal(input)
	if err != nil {
		log.Fatalf("Failed to serialize input: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(inputJSON))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	log.Printf("Invoking workflow at: %s", url)
	log.Printf("Payload: %s", string(inputJSON))
	log.Println("Invoking TutorialWorkflow...")
	client := &http.Client{
		Timeout: 30 * time.Minute, // Long timeout for LLM calls
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to invoke workflow: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Fatalf("Workflow failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse result
	var result types.WriteMarkdownFilesOutput
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Warning: Could not parse result: %v", err)
		log.Printf("Response: %s", string(body))
	} else {
		log.Println("\n✅ Tutorial generated successfully!")
		log.Printf("Files written (%d):", len(result.FilesWritten))
		for _, file := range result.FilesWritten {
			log.Printf("  - %s", file)
		}
	}
}
