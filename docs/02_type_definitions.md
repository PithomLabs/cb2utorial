# Phase 2: Type Definitions

## Goal
Define all shared types for services, workflows, and data structures.

## Decisions Made

**Package**: `types` - Centralized type definitions for all components

**Core Types**:
1. **FileContent** - Represents a file with index, path, content
2. **Abstraction** - Core code concept identified by LLM
3. **Relationship** - How abstractions interact
4. **RelationshipData** - Summary + list of relationships
5. **ChapterSummary** - Brief chapter info for context
6. **WriteChapterOutput** - Generated chapter content

**Input/Output Structs** for each service:
- ReadFilesInput/Output
- AnalyzeAbstractionsInput/Output
- AnalyzeRelationshipsInput
- OrderChaptersInput/Output
- WriteChapterInput/Output
- WriteMarkdownFilesInput/Output
- TutorialWorkflowInput

**Why**:
- Strong typing prevents runtime errors
- JSON tags for Restate serialization
- Self-documenting code
- Compile-time validation

## Design Choices

**Index-based references**: Files and abstractions use integer indices instead of names/paths
- Reduces LLM context size
- Simplifies validation
- Language-agnostic

**Separate Input/Output**: Clear contracts for each service
- Easy to test
- Clear responsibility boundaries

**Embedded types**: ChapterSummary reused in multiple places
- DRY principle
- Consistency

## Next Steps
Phase 3 will implement LLM client wrapper using these types.
