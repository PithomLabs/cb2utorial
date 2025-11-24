package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
)

// FileInfo represents a discovered file
type FileInfo struct {
	RelativePath string
	Content      string
}

// WalkDirectoryOptions configures directory traversal
type WalkDirectoryOptions struct {
	RootPath        string
	IncludePatterns []string
	ExcludePatterns []string
	MaxFileSize     int64
	MaxFiles        int
}

// WalkDirectory traverses a directory and returns matching files
func WalkDirectory(opts WalkDirectoryOptions) ([]FileInfo, error) {
	var files []FileInfo

	// Compile glob patterns
	includeGlobs := make([]glob.Glob, 0, len(opts.IncludePatterns))
	for _, pattern := range opts.IncludePatterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid include pattern %s: %w", pattern, err)
		}
		includeGlobs = append(includeGlobs, g)
	}

	excludeGlobs := make([]glob.Glob, 0, len(opts.ExcludePatterns))
	for _, pattern := range opts.ExcludePatterns {
		g, err := glob.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %s: %w", pattern, err)
		}
		excludeGlobs = append(excludeGlobs, g)
	}

	// Get absolute path for proper relative path calculation
	absRoot, err := filepath.Abs(opts.RootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}

		// Normalize path separators for matching (use forward slash)
		normalizedPath := filepath.ToSlash(relPath)

		// Check exclude patterns first
		for _, g := range excludeGlobs {
			if g.Match(normalizedPath) {
				return nil // Skip this file
			}
		}

		// Check include patterns (if any specified)
		if len(includeGlobs) > 0 {
			matched := false
			for _, g := range includeGlobs {
				if g.Match(normalizedPath) {
					matched = true
					break
				}
			}
			if !matched {
				return nil // Skip this file
			}
		}

		// Check file size limit
		if opts.MaxFileSize > 0 && info.Size() > opts.MaxFileSize {
			return nil // Skip large files
		}

		// Check max files limit
		if opts.MaxFiles > 0 && len(files) >= opts.MaxFiles {
			return filepath.SkipAll // Stop walking
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			// Skip files we can't read (permissions, etc.)
			return nil
		}

		files = append(files, FileInfo{
			RelativePath: normalizedPath,
			Content:      string(content),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return files, nil
}

// SanitizeFilename converts a string into a valid filename
func SanitizeFilename(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces and special chars with underscore
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)

	// Collapse multiple underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	// Trim underscores from edges
	name = strings.Trim(name, "_")

	return name
}
