package handlers

import (
	"net/http"

	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/dione-docs-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShareDocumentRequest struct {
	UserEmail  string `json:"user_email" binding:"required"`
	AccessType string `json:"access_type" binding:"required"`
}

type RemoveAccessRequest struct {
	UserEmail string `json:"user_email" binding:"required"`
}

type PermissionResponse struct {
	ID         uuid.UUID `json:"id"`
	DocumentID uuid.UUID `json:"document_id"`
	UserID     uuid.UUID `json:"user_id"`
	UserEmail  string    `json:"user_email"`
	AccessType string    `json:"access_type"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

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
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgeyi paylaşma izniniz yok"})
			return
		}
	}

	var shareRequest ShareDocumentRequest
	if err := c.ShouldBindJSON(&shareRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz istek formatı"})
		return
	}

	targetUser, err := h.repo.User.GetByEmail(shareRequest.UserEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Kullanıcı bulunamadı"})
		return
	}

	if shareRequest.AccessType != "read" && shareRequest.AccessType != "edit" && shareRequest.AccessType != "admin" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz erişim tipi. 'read', 'edit' veya 'admin' olmalıdır"})
		return
	}

	if targetUser.ID == userID {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Belgeyi kendinizle paylaşamazsınız"})
		return
	}

	existingPermission, err := h.repo.Permission.GetByDocumentAndUser(docID, targetUser.ID)
	if err == nil && existingPermission != nil {
		if err := h.repo.Permission.UpdateAccessType(existingPermission.ID, shareRequest.AccessType); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "İzin güncellenemedi: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, MessageResponse{Message: "Erişim izni güncellendi"})
		return
	}

	permission := &models.Permission{
		DocumentID: docID,
		UserID:     targetUser.ID,
		AccessType: shareRequest.AccessType,
	}

	if err := h.repo.Permission.Create(permission); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "İzin oluşturulamadı: " + err.Error()})
		return
	}

	response := PermissionResponse{
		ID:         permission.ID,
		DocumentID: permission.DocumentID,
		UserID:     permission.UserID,
		UserEmail:  targetUser.Email,
		AccessType: permission.AccessType,
	}

	c.JSON(http.StatusCreated, response)
}

func (h *PermissionHandler) RemoveAccess(c *gin.Context) {
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
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgenin erişimini kaldırma izniniz yok"})
			return
		}
	}

	var removeRequest RemoveAccessRequest
	if err := c.ShouldBindJSON(&removeRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz istek formatı"})
		return
	}

	targetUser, err := h.repo.User.GetByEmail(removeRequest.UserEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Kullanıcı bulunamadı"})
		return
	}

	if err := h.repo.Permission.DeleteByDocumentAndUser(docID, targetUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Erişim kaldırılamadı: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Erişim başarıyla kaldırıldı"})
}

func (h *PermissionHandler) GetDocumentPermissions(c *gin.Context) {
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
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgenin izinlerini görüntüleme yetkiniz yok"})
			return
		}
	}

	permissions, err := h.repo.Permission.GetByDocument(docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "İzinler alınamadı: " + err.Error()})
		return
	}

	var permissionResponses []PermissionResponse
	for _, perm := range permissions {
		user := models.User{}
		err := h.repo.User.GetByID(perm.UserID, &user)
		if err != nil {
			continue
		}

		permissionResponses = append(permissionResponses, PermissionResponse{
			ID:         perm.ID,
			DocumentID: perm.DocumentID,
			UserID:     perm.UserID,
			UserEmail:  user.Email,
			AccessType: perm.AccessType,
		})
	}

	c.JSON(http.StatusOK, permissionResponses)
}
