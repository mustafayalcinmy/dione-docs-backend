package handlers

import (
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

func (h *DocumentHandler) CreateDocument(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kimlik doğrulama hatası"})
		return
	}

	emptyContent := []byte(`{
		"metadata": {
			"title": "Adsız Döküman",
			"author": "` + userID.String() + `",
			"lastModified": "` + time.Now().Format(time.RFC3339) + `"
		},
		"content": [
			{
				"id": "paragraph-1",
				"type": "paragraph",
				"content": ""
			}
		]
	}`)

	doc := &models.Document{
		Title:    "Adsız Döküman",
		OwnerID:  userID,
		Content:  emptyContent,
		Version:  1,
		IsPublic: false,
		Status:   "draft",
	}

	if err := h.repo.Document.Create(doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Belge oluşturulamadı: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, doc)
}

func (h *DocumentHandler) GetDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz belge ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kimlik doğrulama hatası"})
		return
	}

	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Belge bulunamadı"})
		return
	}

	if doc.OwnerID != userID && !doc.IsPublic {
		permission, err := h.repo.Permission.GetByDocumentAndUser(docID, userID)
		if err != nil || permission == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bu belgeye erişim izniniz yok"})
			return
		}
	}

	c.JSON(http.StatusOK, doc)
}

func (h *DocumentHandler) UpdateDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz belge ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kimlik doğrulama hatası"})
		return
	}

	var existingDoc models.Document
	if err := h.repo.Document.GetByID(docID, &existingDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Belge bulunamadı"})
		return
	}

	if existingDoc.OwnerID != userID {
		permission, err := h.repo.Permission.GetByDocumentAndUser(docID, userID)
		if err != nil || permission == nil || (permission.AccessType != "edit" && permission.AccessType != "admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bu belgeyi düzenleme izniniz yok"})
			return
		}
	}

	var updateRequest struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Content     []byte  `json:"content"`
		IsPublic    *bool   `json:"is_public"`
		Status      *string `json:"status"`
	}

	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek formatı"})
		return
	}

	if len(updateRequest.Content) > 0 && string(updateRequest.Content) != string(existingDoc.Content) {
		version := &models.DocumentVersion{
			DocumentID: existingDoc.ID,
			Version:    existingDoc.Version,
			Content:    existingDoc.Content,
			ChangedBy:  userID,
		}
		if err := h.repo.Document.SaveVersion(version); err != nil {
			log.Printf("Versiyon kaydedilemedi: %v", err)
		}
		existingDoc.Version++
	}

	if updateRequest.Title != nil {
		existingDoc.Title = *updateRequest.Title
	}
	if updateRequest.Description != nil {
		existingDoc.Description = *updateRequest.Description
	}
	if len(updateRequest.Content) > 0 {
		existingDoc.Content = updateRequest.Content
	}
	if updateRequest.IsPublic != nil {
		existingDoc.IsPublic = *updateRequest.IsPublic
	}
	if updateRequest.Status != nil {
		existingDoc.Status = *updateRequest.Status
	}

	if err := h.repo.Document.Update(&existingDoc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Belge güncellenemedi: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, existingDoc)
}

func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz belge ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kimlik doğrulama hatası"})
		return
	}

	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Belge bulunamadı"})
		return
	}

	if doc.OwnerID != userID {
		permission, err := h.repo.Permission.GetByDocumentAndUser(docID, userID)
		if err != nil || permission == nil || permission.AccessType != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bu belgeyi silme izniniz yok"})
			return
		}
	}

	if err := h.repo.Document.Delete(&doc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Belge silinemedi: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Belge başarıyla silindi"})
}

func (h *DocumentHandler) GetUserDocuments(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kimlik doğrulama hatası"})
		return
	}

	ownedDocs, err := h.repo.Document.GetByOwnerID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Belgeler alınamadı: " + err.Error()})
		return
	}

	sharedDocs, err := h.repo.Document.GetSharedWithUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Paylaşılan belgeler alınamadı: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"owned":  ownedDocs,
		"shared": sharedDocs,
	})
}

func (h *DocumentHandler) GetDocumentVersions(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz belge ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kimlik doğrulama hatası"})
		return
	}

	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Belge bulunamadı"})
		return
	}

	if doc.OwnerID != userID && !doc.IsPublic {
		permission, err := h.repo.Permission.GetByDocumentAndUser(docID, userID)
		if err != nil || permission == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Bu belgenin geçmişine erişim izniniz yok"})
			return
		}
	}

	versions, err := h.repo.Document.GetVersions(docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Versiyon geçmişi alınamadı: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, versions)
}
