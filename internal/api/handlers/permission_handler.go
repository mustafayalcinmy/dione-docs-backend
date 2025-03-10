package handlers

import (
	"net/http"

	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/dione-docs-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PermissionHandler struct {
	repo *repository.Repository
}

func NewPermissionHandler(repo *repository.Repository) *PermissionHandler {
	return &PermissionHandler{
		repo: repo,
	}
}

func (h *PermissionHandler) ShareDocument(c *gin.Context) {
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
			c.JSON(http.StatusForbidden, gin.H{"error": "Bu belgeyi paylaşma izniniz yok"})
			return
		}
	}

	var shareRequest struct {
		UserEmail  string `json:"user_email" binding:"required"`
		AccessType string `json:"access_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&shareRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek formatı"})
		return
	}

	targetUser, err := h.repo.User.GetByEmail(shareRequest.UserEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kullanıcı bulunamadı"})
		return
	}

	if shareRequest.AccessType != "read" && shareRequest.AccessType != "edit" && shareRequest.AccessType != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz erişim tipi. 'read', 'edit' veya 'admin' olmalıdır"})
		return
	}

	if targetUser.ID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Belgeyi kendinizle paylaşamazsınız"})
		return
	}

	existingPermission, err := h.repo.Permission.GetByDocumentAndUser(docID, targetUser.ID)
	if err == nil && existingPermission != nil {
		if err := h.repo.Permission.UpdateAccessType(existingPermission.ID, shareRequest.AccessType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "İzin güncellenemedi: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Erişim izni güncellendi"})
		return
	}

	permission := &models.Permission{
		DocumentID: docID,
		UserID:     targetUser.ID,
		AccessType: shareRequest.AccessType,
	}

	if err := h.repo.Permission.Create(permission); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "İzin oluşturulamadı: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, permission)
}

func (h *PermissionHandler) RemoveAccess(c *gin.Context) {
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
			c.JSON(http.StatusForbidden, gin.H{"error": "Bu belgenin erişimini kaldırma izniniz yok"})
			return
		}
	}

	var removeRequest struct {
		UserEmail string `json:"user_email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&removeRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Geçersiz istek formatı"})
		return
	}

	targetUser, err := h.repo.User.GetByEmail(removeRequest.UserEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kullanıcı bulunamadı"})
		return
	}

	if err := h.repo.Permission.DeleteByDocumentAndUser(docID, targetUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erişim kaldırılamadı: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Erişim başarıyla kaldırıldı"})
}

func (h *PermissionHandler) GetDocumentPermissions(c *gin.Context) {
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
			c.JSON(http.StatusForbidden, gin.H{"error": "Bu belgenin izinlerini görüntüleme yetkiniz yok"})
			return
		}
	}

	permissions, err := h.repo.Permission.GetByDocument(docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "İzinler alınamadı: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, permissions)
}
