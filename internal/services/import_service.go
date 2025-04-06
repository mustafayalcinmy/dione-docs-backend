package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log" // Make sure log package is imported

	"github.com/dione-docs-backend/internal/models"     // Adjust import path
	"github.com/dione-docs-backend/internal/parser"     // Adjust import path
	"github.com/dione-docs-backend/internal/repository" // Adjust import path
	"github.com/google/uuid"
)

// ImportService handles the logic for importing documents.
type ImportService struct {
	docRepo repository.DocumentRepository
	// Use the specific parser type or the interface
	// Example using the interface:
	theParser parser.Parser
	// Example using concrete type if not using interface:
	// theParser *docx.manualParser
}

// NewImportService creates a new ImportService.
func NewImportService(docRepo repository.DocumentRepository, p parser.Parser) *ImportService {
	return &ImportService{
		docRepo:   docRepo,
		theParser: p,
	}
}

// ImportDocument orchestrates the document import process.
// Modify signature based on how you handle file path vs reader
func (s *ImportService) ImportDocument(ctx context.Context, userID uuid.UUID, reader io.ReaderAt, size int64, fileType string, originalFilename string) (*models.Document, error) {

	var parsedData *models.ParsedContent
	var err error

	// --- Step 1: Call the Parser ---
	log.Printf("Calling parser for file: %s", originalFilename)
	switch fileType {
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		// --- Use the correct arguments for the manual parser ---
		parsedData, err = s.theParser.Parse(reader, size) // <-- Corrected this line
	default:
		return nil, fmt.Errorf("unsupported file type for import: %s", fileType)
	}

	if err != nil {
		// Make sure parser errors are logged/returned clearly
		log.Printf("!!! Parser returned error: %v", err)
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}
	if parsedData == nil {
		// Parser might succeed but return nil data
		log.Printf("!!! Parser returned nil data")
		return nil, fmt.Errorf("parser returned nil data")
	}

	// --- Step 2: Check and Log Parser Output ---
	// Check if the parser actually returned content. The JSON log added
	// previously in the parser itself should also show this.
	if len(parsedData.Content) == 0 {
		log.Println("!!! Warning: Parser returned ParsedContent with zero Content nodes.")
	} else {
		log.Printf("--- Parser returned ParsedContent with %d top-level nodes ---", len(parsedData.Content))
	}

	// --- Step 3: Marshal Parsed Data and Log Result ---
	log.Println("--- Marshaling ParsedContent to JSON ---")
	contentBytes, err := json.Marshal(parsedData.Content)
	if err != nil {
		// Log the error clearly if marshaling fails
		log.Printf("!!! Error marshaling ParsedContent: %v", err)
		return nil, fmt.Errorf("failed to marshal parsed content to JSON: %w", err)
	}
	// Log the marshaled bytes (or their length)
	log.Printf("--- Marshaled JSON Bytes Length: %d ---", len(contentBytes))
	if len(contentBytes) < 500 { // Log short content for inspection
		log.Printf("--- Marshaled JSON Bytes Content: %s ---", string(contentBytes))
	}
	if len(contentBytes) == 0 || string(contentBytes) == "null" || string(contentBytes) == "{}" {
		log.Println("!!! Warning: Marshaled content bytes are empty or represent empty/null JSON!")
	}

	// --- Step 4: Prepare Document Model ---
	log.Println("--- Preparing Document model for database ---")
	// TODO: Extract title better if possible
	title := originalFilename // Placeholder

	doc := &models.Document{
		Title:       title,
		Description: "Imported document",
		OwnerID:     userID,
		Content:     contentBytes, // Assign the marshaled bytes
		Version:     1,
		IsPublic:    false,
		Status:      "draft",
	}

	// --- Step 5: Log Document Just Before Saving ---
	previewLen := 100
	if len(doc.Content) < previewLen {
		previewLen = len(doc.Content)
	}
	log.Printf("--- Document content length before save: %d ---", len(doc.Content))
	log.Printf("--- Document content preview before save: %s ---", string(doc.Content[:previewLen]))
	if len(doc.Content) == 0 {
		log.Println("!!! Error: Document Content field is empty before calling Create!")
	}

	// --- Step 6: Save to Database and Check Error ---
	log.Println("--- Calling docRepo.Create ---")
	if err := s.docRepo.Create(doc); err != nil {
		// Log the specific error from the database operation
		log.Printf("!!! Error calling docRepo.Create: %v", err)
		return nil, fmt.Errorf("failed to save imported document: %w", err)
	}

	log.Println("--- Document saved successfully (ID: %s) ---", doc.ID)
	return doc, nil
}
