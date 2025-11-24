package services

import (
	"fmt"

	"github.com/pithomlabs/cb2utorial/types"
	"github.com/pithomlabs/cb2utorial/utils"
	restate "github.com/restatedev/sdk-go"
)

// FileReaderService reads files from a local directory
type FileReaderService struct{}

// ServiceName returns the service name for registration
func (s FileReaderService) ServiceName() string {
	return "FileReader"
}

// ReadFiles traverses the local repository and returns indexed file list
func (s FileReaderService) ReadFiles(ctx restate.Context, input types.ReadFilesInput) (types.ReadFilesOutput, error) {
	// Validate input
	if input.RepoPath == "" {
		return types.ReadFilesOutput{}, fmt.Errorf("repo_path is required")
	}

	// Walk directory with configured options
	fileInfos, err := utils.WalkDirectory(utils.WalkDirectoryOptions{
		RootPath:        input.RepoPath,
		IncludePatterns: input.IncludePatterns,
		ExcludePatterns: input.ExcludePatterns,
		MaxFileSize:     input.MaxFileSize,
		MaxFiles:        input.MaxFiles,
	})
	if err != nil {
		return types.ReadFilesOutput{}, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Convert to indexed FileContent list
	files := make([]types.FileContent, len(fileInfos))
	for i, info := range fileInfos {
		files[i] = types.FileContent{
			Index:   i,
			Path:    info.RelativePath,
			Content: info.Content,
		}
	}

	return types.ReadFilesOutput{
		Files: files,
	}, nil
}
