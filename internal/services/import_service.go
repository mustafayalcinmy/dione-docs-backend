package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/dione-docs-backend/internal/config"
	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/google/uuid"
)

type ImportService struct {
	docRepo repository.DocumentRepository
	cfg     *config.Config
}

func NewImportService(docRepo repository.DocumentRepository, cfg *config.Config) *ImportService {
	return &ImportService{
		docRepo: docRepo,
		cfg:     cfg,
	}
}

func (s *ImportService) ImportDocument(ctx context.Context, userID uuid.UUID, fileReader io.Reader, originalFilename string) (*models.Document, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", originalFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, fileReader); err != nil {
		return nil, fmt.Errorf("failed to copy file content to form: %w", err)
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", s.cfg.PythonServiceURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for python service: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to python service: %w", err)
	}
	defer resp.Body.Close()

	// Python'dan gelen yanıtı oku.
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from python service: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("python service returned non-200 status: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	finalContentForDB, err := json.Marshal(string(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create the final escaped-json string: %w", err)
	}

	log.Printf("--- Successfully created the final 'escaped string' format for DB. ---")

	doc := &models.Document{
		Title:       originalFilename,
		Description: "Imported document from " + originalFilename,
		OwnerID:     userID,
		Content:     finalContentForDB,
		Version:     1,
		IsPublic:    false,
		Status:      "draft",
	}

	if err := s.docRepo.Create(doc); err != nil {
		return nil, fmt.Errorf("failed to save imported document: %w", err)
	}

	log.Printf("--- Document saved successfully (ID: %s) ---", doc.ID)
	return doc, nil
}
