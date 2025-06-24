package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dione-docs-backend/internal/services"
	"github.com/dione-docs-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

type ImportHandler struct {
	importService *services.ImportService
}

func NewImportHandler(importService *services.ImportService) *ImportHandler {
	return &ImportHandler{
		importService: importService,
	}
}

// ImportDocxHandler handles the .docx file upload and import request.
// @Tags         Documents
// @Summary      Import a DOCX document
// @Description  Uploads a DOCX file, sends it to an external Python service for conversion to Quill Delta format, and creates a new document.
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "DOCX file to import"
// @Success      201  {object}  DocumentResponse  "Document imported successfully"
// @Failure      400  {object}  ErrorResponse     "Bad request (e.g., no file)"
// @Failure      401  {object}  ErrorResponse     "Authentication error"
// @Failure      500  {object}  ErrorResponse     "Internal server error (e.g., Python service unavailable, parsing or saving failed)"
// @Router       /api/v1/import/docx [post]
func (h *ImportHandler) ImportDocxHandler(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authentication required"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "File upload error: " + err.Error()})
		return
	}

	// Yüklenecek dosyayı aç
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to open uploaded file"})
		return
	}
	defer file.Close()

	createdDoc, err := h.importService.ImportDocument(c.Request.Context(), userID, file, fileHeader.Filename)
	if err != nil {
		log.Printf("Error importing document via python service: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("Failed to import document: %v", err)})
		return
	}

	response := documentToResponse(createdDoc)
	c.JSON(http.StatusCreated, response)
}
