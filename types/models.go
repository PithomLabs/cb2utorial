package types

// FileContent represents a source code file with indexed reference
type FileContent struct {
	Index   int    `json:"index"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Abstraction represents a core code concept identified by LLM
type Abstraction struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FileIndices []int  `json:"file_indices"` // References to FileContent by index
}

// Relationship describes how two abstractions interact
type Relationship struct {
	FromIndex int    `json:"from_index"` // Source abstraction index
	ToIndex   int    `json:"to_index"`   // Target abstraction index
	Label     string `json:"label"`      // Interaction description
}

// RelationshipData contains project summary and abstraction relationships
type RelationshipData struct {
	Summary string         `json:"summary"`
	Details []Relationship `json:"details"`
}

// ChapterSummary provides brief context about a chapter
type ChapterSummary struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

// WriteChapterOutput contains generated chapter content
type WriteChapterOutput struct {
	ChapterNumber int    `json:"chapter_number"`
	Title         string `json:"title"`
	Content       string `json:"content"` // Markdown content
}

// ===== Service Input/Output Types =====

// ReadFilesInput configures file reading from local directory
type ReadFilesInput struct {
	RepoPath        string   `json:"repo_path"`
	IncludePatterns []string `json:"include_patterns"`
	ExcludePatterns []string `json:"exclude_patterns"`
	MaxFileSize     int64    `json:"max_file_size"`
	MaxFiles        int      `json:"max_files"`
}

// ReadFilesOutput returns indexed file list
type ReadFilesOutput struct {
	Files []FileContent `json:"files"`
}

// AnalyzeAbstractionsInput provides codebase for abstraction analysis
type AnalyzeAbstractionsInput struct {
	Files           []FileContent `json:"files"`
	ProjectName     string        `json:"project_name"`
	MaxAbstractions int           `json:"max_abstractions"`
}

// AnalyzeAbstractionsOutput returns identified abstractions
type AnalyzeAbstractionsOutput struct {
	Abstractions []Abstraction `json:"abstractions"`
}

// AnalyzeRelationshipsInput provides context for relationship analysis
type AnalyzeRelationshipsInput struct {
	Abstractions []Abstraction `json:"abstractions"`
	Files        []FileContent `json:"files"`
	ProjectName  string        `json:"project_name"`
}

// OrderChaptersInput provides data for determining chapter sequence
type OrderChaptersInput struct {
	Abstractions  []Abstraction    `json:"abstractions"`
	Relationships RelationshipData `json:"relationships"`
	ProjectName   string           `json:"project_name"`
}

// OrderChaptersOutput returns pedagogically-ordered abstraction indices
type OrderChaptersOutput struct {
	OrderedIndices []int `json:"ordered_indices"`
}

// WriteChapterInput provides context for writing a single chapter
type WriteChapterInput struct {
	Abstraction      Abstraction      `json:"abstraction"`
	Files            []FileContent    `json:"files"`
	PreviousChapters []ChapterSummary `json:"previous_chapters"`
	ProjectName      string           `json:"project_name"`
	ChapterNumber    int              `json:"chapter_number"`
}

// WriteMarkdownFilesInput specifies where to write chapters
type WriteMarkdownFilesInput struct {
	OutputDir string               `json:"output_dir"`
	Chapters  []WriteChapterOutput `json:"chapters"`
}

// WriteMarkdownFilesOutput returns paths of created files
type WriteMarkdownFilesOutput struct {
	FilesWritten []string `json:"files_written"`
}

// TutorialWorkflowInput configures the entire tutorial generation workflow
type TutorialWorkflowInput struct {
	LocalRepoPath string `json:"local_repo_path"`
	OutputDir     string `json:"output_dir"`
	MaxFiles      int    `json:"max_files"`
	ProjectName   string `json:"project_name,omitempty"` // Optional, derived from path if empty
}

// TutorialState tracks workflow progress (stored in workflow context)
type TutorialState struct {
	Files           []FileContent        `json:"files"`
	Abstractions    []Abstraction        `json:"abstractions"`
	Relationships   RelationshipData     `json:"relationships"`
	ChapterOrder    []int                `json:"chapter_order"`
	Chapters        []WriteChapterOutput `json:"chapters"`
	ChaptersWritten []string             `json:"chapters_written"` // For cleanup
}
