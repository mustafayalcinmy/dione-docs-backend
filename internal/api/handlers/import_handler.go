package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"

	// Adjust import paths as needed
	"github.com/dione-docs-backend/internal/services"
	"github.com/dione-docs-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// ImportHandler handles requests related to document importing.
type ImportHandler struct {
	importService *services.ImportService
	// Add other dependencies like config if needed
}

// NewImportHandler creates a new ImportHandler.
func NewImportHandler(importService *services.ImportService) *ImportHandler {
	return &ImportHandler{
		importService: importService,
	}
}

// ImportDocxHandler handles the .docx file upload and import request.
// @Tags Documents
// @Summary Import a DOCX document
// @Description Uploads a DOCX file and converts it into a new document.
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "DOCX file to import"
// @Success 201 {object} DocumentResponse "Document imported successfully"
// @Failure 400 {object} ErrorResponse "Bad request (e.g., no file, wrong type)"
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 500 {object} ErrorResponse "Internal server error (parsing, saving)"
// @Router /api/v1/import/docx [post] // Example route
func (h *ImportHandler) ImportDocxHandler(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c) // Assuming JWT middleware sets this
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authentication required"})
		return
	}

	fileHeader, err := c.FormFile("file") // "file" is the name of the form field
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "File upload error: " + err.Error()})
		return
	}

	// Optional: Check file extension or MIME type early
	// fileType := fileHeader.Header.Get("Content-Type")
	// if fileType != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
	//  c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid file type, only .docx is supported"})
	//  return
	// }

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to open uploaded file"})
		return
	}
	defer file.Close()

	// Use the import service to process the file
	// Note: We need an io.ReaderAt for parsing libraries that expect seeking (like zip readers)
	// Gin's file implements io.Reader, io.Seeker, io.Closer - perfect for io.ReaderAt
	readerAt, ok := file.(io.ReaderAt)
	if !ok {
		// This should ideally not happen with Gin's uploaded file type
		log.Printf("Warning: Uploaded file does not implement io.ReaderAt")
		// Fallback might involve reading into memory, which is bad for large files
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Internal error processing file type"})
		return
	}

	// Get file size
	fileSize := fileHeader.Size

	// Get MIME type for parser selection in service
	fileType := fileHeader.Header.Get("Content-Type")

	createdDoc, err := h.importService.ImportDocument(c.Request.Context(), userID, readerAt, fileSize, fileType, fileHeader.Filename)
	println("Created Document:", createdDoc.Content) // Debugging line
	if err != nil {
		log.Printf("Error importing document: %v", err)
		// TODO: Provide more specific error messages based on err type (parsing vs saving)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("Failed to import document: %v", err)})
		return
	}

	// Return the created document details (using existing DocumentResponse)
	// TODO: Ensure documentToResponse handles the new content structure or omit content
	response := documentToResponse(createdDoc) // Reuse your existing response converter
	c.JSON(http.StatusCreated, response)
}
