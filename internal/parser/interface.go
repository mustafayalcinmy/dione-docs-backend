package parser

import (
	"io"

	"github.com/dione-docs-backend/internal/models" // Adjust import path if needed
)

// Parser defines the interface for parsing different file formats.
type Parser interface {
	// Parse takes a reader containing the file content and returns the structured content
	// or an error if parsing fails.
	Parse(reader io.ReaderAt, size int64) (*models.ParsedContent, error)
}
