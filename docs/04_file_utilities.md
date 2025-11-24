# Phase 4: File Utilities

## Goal
Implement file reading (local directory traversal) and writing services.

## Components

1. **utils/filewalker.go** - File system utilities
   - `WalkDirectory()` - Traverse directory with pattern matching
   - Uses `filepath.Walk` and `github.com/gobwas/glob`

2. **services/file_reader.go** - FileReaderService (Restate service)
   - Reads local repository files
   - Filters by patterns, size limits
   - Returns indexed FileContent list

3. **services/file_writer.go** - FileWriterService (Restate service)
   - Writes markdown chapters to disk
   - Creates output directory
   - Returns list of written file paths

## Design Decisions

**Glob Pattern Matching**: Using `github.com/gobwas/glob` for flexible patterns
- Supports wildcards: `*.go`, `**/*.py`
- Efficient compiled matchers

**File Size Limits**: Skip files larger than MAX_FILE_SIZE
- Prevents memory issues with large binaries
- Configurable via environment

**Index Assignment**: Files indexed sequentially during traversal
- Consistent ordering
- Deterministic results

**Relative Paths**: Store relative paths from repo root
- Portability
- Cleaner output

**Filename Sanitization**: Convert abstraction names to valid filenames
- Remove special characters
- Lowercase with underscores
- Example: "Node Abstraction" → "node_abstraction"

## Restate Service Pattern

Both services use standard `restate.Context`:
- Stateless services (data plane)
- No state persistence needed
- Pure input → output transformation

## Next Steps
Phase 5 will implement analysis services (AbstractionAnalyzer, RelationshipAnalyzer, ChapterOrderer).
