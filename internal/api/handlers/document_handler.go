package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/dione-docs-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DocumentHandler struct {
	repo *repository.Repository
}

func NewDocumentHandler(repo *repository.Repository) *DocumentHandler {
	return &DocumentHandler{
		repo: repo,
	}
}

type DocumentResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	OwnerID     uuid.UUID `json:"owner_id"`
	Version     int       `json:"version"`
	IsPublic    bool      `json:"is_public"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Content     []byte    `json:"content,omitempty"`
}

type DocumentListResponse struct {
	Owned  []DocumentResponse `json:"owned"`
	Shared []DocumentResponse `json:"shared"`
}

type DocumentVersionResponse struct {
	ID         uuid.UUID `json:"id"`
	DocumentID uuid.UUID `json:"document_id"`
	Version    int       `json:"version"`
	ChangedBy  uuid.UUID `json:"changed_by"`
	CreatedAt  time.Time `json:"created_at"`
	Content    []byte    `json:"content,omitempty"`
}

type CreateDocumentRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	IsPublic    bool            `json:"is_public"`
	Content     json.RawMessage `json:"content,omitempty"` // Quill Delta formatında içerik
}

type UpdateDocumentRequest struct {
	Title       *string         `json:"title"`
	Description *string         `json:"description"`
	Content     json.RawMessage `json:"content,omitempty"` // Değişiklik: []byte -> json.RawMessage, omitempty eklendi
	IsPublic    *bool           `json:"is_public"`
	Status      *string         `json:"status"`
}

func documentToResponse(doc *models.Document) DocumentResponse {
	return DocumentResponse{
		ID:          doc.ID,
		Title:       doc.Title,
		Description: doc.Description,
		OwnerID:     doc.OwnerID,
		Version:     doc.Version,
		IsPublic:    doc.IsPublic,
		Status:      doc.Status,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
		Content:     doc.Content,
	}
}

func documentsToResponses(docs []models.Document) []DocumentResponse {
	responses := make([]DocumentResponse, len(docs))
	for i, doc := range docs {
		responses[i] = documentToResponse(&doc)
	}
	return responses
}

func versionToResponse(version *models.DocumentVersion) DocumentVersionResponse {
	return DocumentVersionResponse{
		ID:         version.ID,
		DocumentID: version.DocumentID,
		Version:    version.Version,
		ChangedBy:  version.ChangedBy,
		CreatedAt:  version.CreatedAt,
		Content:    version.Content,
	}
}

func versionsToResponses(versions []models.DocumentVersion) []DocumentVersionResponse {
	responses := make([]DocumentVersionResponse, len(versions))
	for i, version := range versions {
		responses[i] = versionToResponse(&version)
	}
	return responses
}

func getBodyBytes(c *gin.Context) []byte {
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil {
			// Body'yi tekrar okunabilir hale getirmek önemli
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			return bodyBytes
		}
	}
	return nil
}

// CreateDocument creates a new document
// @Tags Documents
// @Summary Create a new document
// @Description Create a new document with title, description, and content
// @Accept  json
// @Produce  json
// @Param document body CreateDocumentRequest true "Document Data"
// @Success 201 {object} DocumentResponse "Document created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request format"
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/documents [post]
func (h *DocumentHandler) CreateDocument(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var request CreateDocumentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		// Hatanın detayını loglayalım
		log.Printf("CreateDocument - ShouldBindJSON error: %v. Gelen veri: %s", err, string(getBodyBytes(c))) // Gelen veriyi logla
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz istek formatı"})
		return
	}

	log.Printf("CreateDocument - Request bind edildi: %+v", request)
	log.Printf("CreateDocument - Request.Content (json.RawMessage): %s", string(request.Content))

	title := "Adsız Döküman"
	if request.Title != "" {
		title = request.Title
	}

	var contentToSave []byte
	if len(request.Content) > 0 && string(request.Content) != "null" { // "null" string'ini de kontrol et
		contentToSave = request.Content // Doğrudan ata, zaten json.RawMessage []byte'tır
	} else {
		contentToSave = []byte(`{"ops":[{"insert":"\n"}]}`)
	}

	doc := &models.Document{
		Title:       title,
		Description: request.Description,
		OwnerID:     userID,
		Content:     contentToSave,
		Version:     1,
		IsPublic:    request.IsPublic,
		Status:      "draft", // Status backend'de atanıyor, request'ten değil
	}

	if err := h.repo.Document.Create(doc); err != nil {
		log.Printf("CreateDocument - repo.Document.Create error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Belge oluşturulamadı: " + err.Error()})
		return
	}
	log.Printf("CreateDocument - Belge başarıyla oluşturuldu: ID %s", doc.ID.String())
	c.JSON(http.StatusCreated, documentToResponse(doc))
}

// GetDocument retrieves a document by its ID
// @Tags Documents
// @Summary Get a document by ID
// @Description Retrieve a document by its unique identifier
// @Produce  json
// @Param id path string true "Document ID"
// @Success 200 {object} DocumentResponse "Document retrieved successfully"
// @Failure 400 {object} ErrorResponse "Invalid document ID"
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Router /api/v1/documents/{id} [get]
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz belge ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Belge bulunamadı"})
		return
	}

	if doc.OwnerID != userID && !doc.IsPublic {
		permission, err := h.repo.Permission.GetByDocumentAndUser(docID, userID)
		if err != nil || permission == nil {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgeye erişim izniniz yok"})
			return
		}
	}

	c.JSON(http.StatusOK, documentToResponse(&doc))
}

// UpdateDocument updates an existing document
// @Tags Documents
// @Summary Update an existing document
// @Description Update the title, description, or content of an existing document
// @Accept  json
// @Produce  json
// @Param id path string true "Document ID"
// @Param document body UpdateDocumentRequest true "Updated document data"
// @Success 200 {object} DocumentResponse "Document updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid document ID or request format"
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/documents/{id} [put]
// Path: dione-docs-backend/internal/api/handlers/document_handler.go
func (h *DocumentHandler) UpdateDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz belge ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var existingDoc models.Document
	if err := h.repo.Document.GetByID(docID, &existingDoc); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Belge bulunamadı"})
		return
	}

	permission := &models.Permission{}
	permission, _ = h.repo.Permission.GetAcceptedByDocumentAndUser(docID, userID)

	if permission.AccessType != string(models.AccessTypeEditor) && permission.AccessType != string(models.AccessTypeAdmin) {
		permission, err := h.repo.Permission.GetByDocumentAndUser(docID, userID)
		if err != nil || permission == nil || (permission.AccessType != "edit" && permission.AccessType != "admin") {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgeyi düzenleme izniniz yok"})
			return
		}
	}

	var updateRequest UpdateDocumentRequest
	// Bind edilecek verinin bir kopyasını alalım, çünkü c.Request.Body sadece bir kez okunabilir.
	var requestBodyBytes []byte
	if c.Request.Body != nil {
		requestBodyBytes, _ = io.ReadAll(c.Request.Body)
	}
	// Orijinal body'yi tekrar yerine koyalım ki ShouldBindJSON okuyabilsin
	c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBodyBytes))

	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		log.Printf("UpdateDocument - ShouldBindJSON error: %v. Gelen veri: %s", err, string(requestBodyBytes))
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz istek formatı: " + err.Error()})
		return
	}

	log.Printf("UpdateDocument - Request bind edildi: %+v", updateRequest)
	if updateRequest.Content != nil {
		log.Printf("UpdateDocument - Request.Content (json.RawMessage): %s", string(updateRequest.Content))
	}

	contentChanged := false
	if updateRequest.Content != nil && len(updateRequest.Content) > 0 && string(updateRequest.Content) != "null" {
		// Gelen json.RawMessage (yani []byte) ile mevcut []byte'ı karşılaştır.
		// Eğer frontend boş bir delta için "{}" veya "{\"ops\":[]}" gibi bir şey gönderiyorsa,
		// ve existingDoc.Content de benzer bir yapıdaysa, bu karşılaştırma doğru çalışmayabilir.
		// Daha sağlam bir karşılaştırma için, her ikisini de unmarshal edip karşılaştırmak gerekebilir,
		// ya da frontend'in "değişiklik yok" durumunda content'i göndermemesi sağlanabilir (`omitempty` sayesinde).
		// Şimdilik basit byte karşılaştırması yapıyoruz.
		if !bytes.Equal(updateRequest.Content, existingDoc.Content) {
			contentChanged = true
		}
	}

	// Save version if content changed
	if contentChanged {
		version := &models.DocumentVersion{
			DocumentID: existingDoc.ID,
			Version:    existingDoc.Version,
			Content:    existingDoc.Content, // Önceki içeriği kaydet
			ChangedBy:  userID,
		}
		if err := h.repo.Document.SaveVersion(version); err != nil {
			log.Printf("Versiyon kaydedilemedi: %v", err)
		}
		existingDoc.Version++
		existingDoc.Content = updateRequest.Content // Yeni içeriği ata (json.RawMessage zaten []byte)
	}

	if updateRequest.Title != nil {
		existingDoc.Title = *updateRequest.Title
	}
	if updateRequest.Description != nil {
		existingDoc.Description = *updateRequest.Description
	}
	// Content güncellemesi yukarıda contentChanged bloğunda yapıldı.
	// Eğer content gönderilmediyse (omitempty sayesinde updateRequest.Content nil ise) veya aynıysa,
	// existingDoc.Content değiştirilmeyecek.
	// Eğer content alanı zorunluysa ve her zaman güncellenmesi gerekiyorsa, bu mantık değişmeli.
	// Mevcut durumda content'i sadece değişmişse güncelliyoruz.
	// Eğer frontend her zaman content gönderiyorsa (boş delta bile olsa),
	// ve `omitempty` kullanılmıyorsa, o zaman aşağıdaki gibi direkt atama yapılabilir:
	// if updateRequest.Content != nil {
	// 	existingDoc.Content = updateRequest.Content
	// }

	if updateRequest.IsPublic != nil {
		existingDoc.IsPublic = *updateRequest.IsPublic
	}
	if updateRequest.Status != nil {
		existingDoc.Status = *updateRequest.Status
	}

	if err := h.repo.Document.Update(&existingDoc); err != nil {
		log.Printf("UpdateDocument - repo.Document.Update error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Belge güncellenemedi: " + err.Error()})
		return
	}
	log.Printf("UpdateDocument - Belge başarıyla güncellendi: ID %s", existingDoc.ID.String())
	c.JSON(http.StatusOK, documentToResponse(&existingDoc))
}

// DeleteDocument deletes a document
// @Tags Documents
// @Summary Delete a document by ID
// @Description Delete an existing document by its unique identifier
// @Param id path string true "Document ID"
// @Success 200 {object} MessageResponse "Document deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid document ID"
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/documents/{id} [delete]
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz belge ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Belge bulunamadı"})
		return
	}

	if doc.OwnerID != userID {
		permission, err := h.repo.Permission.GetByDocumentAndUser(docID, userID)
		if err != nil || permission == nil || permission.AccessType != "admin" {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgeyi silme izniniz yok"})
			return
		}
	}

	if err := h.repo.Document.Delete(&doc); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Belge silinemedi: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Belge başarıyla silindi"})
}

// GetUserDocuments retrieves all documents for the authenticated user
// @Tags Documents
// @Summary Get all documents for the authenticated user
// @Description Retrieve all owned and shared documents for the authenticated user
// @Produce  json
// @Success 200 {object} DocumentListResponse "Documents retrieved successfully"
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/documents/user [get]
func (h *DocumentHandler) GetUserDocuments(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	ownedDocs, err := h.repo.Document.GetByOwnerID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Belgeler alınamadı: " + err.Error()})
		return
	}

	sharedDocs, err := h.repo.Document.GetSharedWithUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Paylaşılan belgeler alınamadı: " + err.Error()})
		return
	}

	response := DocumentListResponse{
		Owned:  documentsToResponses(ownedDocs),
		Shared: documentsToResponses(sharedDocs),
	}

	c.JSON(http.StatusOK, response)
}

// GetDocumentVersions retrieves the version history of a document
// @Tags Documents
// @Summary Get version history of a document
// @Description Retrieve all versions of a specific document
// @Param id path string true "Document ID"
// @Success 200 {array} DocumentVersionResponse "Document version history"
// @Failure 400 {object} ErrorResponse "Invalid document ID"
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Document not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/documents/{id}/versions [get]
func (h *DocumentHandler) GetDocumentVersions(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz belge ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Belge bulunamadı"})
		return
	}

	if doc.OwnerID != userID && !doc.IsPublic {
		permission, err := h.repo.Permission.GetByDocumentAndUser(docID, userID)
		if err != nil || permission == nil {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgenin geçmişine erişim izniniz yok"})
			return
		}
	}

	versions, err := h.repo.Document.GetVersions(docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Versiyon geçmişi alınamadı: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, versionsToResponses(versions))
}
