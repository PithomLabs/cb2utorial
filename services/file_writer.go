package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pithomlabs/cb2utorial/types"
	"github.com/pithomlabs/cb2utorial/utils"
	restate "github.com/restatedev/sdk-go"
)

// FileWriterService writes markdown files to disk
type FileWriterService struct{}

// ServiceName returns the service name for registration
func (s FileWriterService) ServiceName() string {
	return "FileWriter"
}

// WriteMarkdownFiles creates chapter files from generated content
func (s FileWriterService) WriteMarkdownFiles(ctx restate.Context, input types.WriteMarkdownFilesInput) (types.WriteMarkdownFilesOutput, error) {
	// Validate input
	if input.OutputDir == "" {
		return types.WriteMarkdownFilesOutput{}, fmt.Errorf("output_dir is required")
	}

	// Create output directory if it doesn't exist
	err := os.MkdirAll(input.OutputDir, 0755)
	if err != nil {
		return types.WriteMarkdownFilesOutput{}, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write each chapter to a file
	var filesWritten []string

	for _, chapter := range input.Chapters {
		// Sanitize title for filename
		sanitizedTitle := utils.SanitizeFilename(chapter.Title)

		// Format filename with chapter number
		filename := fmt.Sprintf("%02d_%s.md", chapter.ChapterNumber, sanitizedTitle)
		filePath := filepath.Join(input.OutputDir, filename)

		// Write markdown content
		err := os.WriteFile(filePath, []byte(chapter.Content), 0644)
		if err != nil {
			return types.WriteMarkdownFilesOutput{}, fmt.Errorf("failed to write file %s: %w", filename, err)
		}

		filesWritten = append(filesWritten, filePath)
	}

	return types.WriteMarkdownFilesOutput{
		FilesWritten: filesWritten,
	}, nil
}
