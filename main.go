package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/pithomlabs/cb2utorial/services"
	"github.com/pithomlabs/cb2utorial/workflow"
	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/server"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Validate required environment variables
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}

	// Create Restate server and bind all services using method chaining
	// Note: Bind() returns *Restate for chaining, not an error
	server := server.NewRestate().
		Bind(restate.Reflect(services.FileReaderService{})).
		Bind(restate.Reflect(services.AbstractionAnalyzerService{})).
		Bind(restate.Reflect(services.RelationshipAnalyzerService{})).
		Bind(restate.Reflect(services.ChapterOrdererService{})).
		Bind(restate.Reflect(services.ChapterWriterService{})).
		Bind(restate.Reflect(services.FileWriterService{})).
		Bind(restate.Reflect(workflow.TutorialWorkflow{}))

	log.Println("Starting Restate server on :9082...")
	log.Println("Services registered:")
	log.Println("  - FileReader")
	log.Println("  - AbstractionAnalyzer")
	log.Println("  - RelationshipAnalyzer")
	log.Println("  - ChapterOrderer")
	log.Println("  - ChapterWriter")
	log.Println("  - FileWriter")
	log.Println("Workflows registered:")
	log.Println("  - TutorialWorkflow")

	// Start server
	if err := server.Start(context.Background(), ":9082"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
