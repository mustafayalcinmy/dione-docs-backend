package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/dione-docs-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ShareDocumentRequest UserEmail ve AccessType alanlarını içerir.
type ShareDocumentRequest struct {
	UserEmail  string `json:"user_email" binding:"required,email"`
	AccessType string `json:"access_type" binding:"required"` // "viewer" veya "editor" olmalı
}

// RemoveAccessRequest UserEmail alanını içerir.
type RemoveAccessRequest struct {
	UserEmail string `json:"user_email" binding:"required,email"`
}

// PermissionResponse, ID, DocumentID, UserID, UserEmail ve AccessType alanlarını içerir.
// Davetiyeler için buna Status ve SharedByEmail gibi alanlar eklenebilir.
type PermissionResponse struct {
	ID         uuid.UUID               `json:"id"`
	DocumentID uuid.UUID               `json:"document_id"`
	UserID     uuid.UUID               `json:"user_id"`
	UserEmail  string                  `json:"user_email"`
	AccessType string                  `json:"access_type"`
	Status     models.PermissionStatus `json:"status"`              // Eklendi
	SharedBy   string                  `json:"shared_by,omitempty"` // Daveti gönderenin e-postası (opsiyonel)
	// DocumentTitle string            `json:"document_title,omitempty"` // Davetiyeler listelenirken gerekebilir
}

// InvitationDetailResponse, davetiyeleri listelerken kullanılabilir.
type InvitationDetailResponse struct {
	InvitationID  uuid.UUID               `json:"invitation_id"` // Permission ID
	DocumentID    uuid.UUID               `json:"document_id"`
	DocumentTitle string                  `json:"document_title"`
	SharedByEmail string                  `json:"shared_by_email"`
	AccessType    string                  `json:"access_type"`
	Status        models.PermissionStatus `json:"status"`
	CreatedAt     time.Time               `json:"created_at"`
}

// MessageResponse, genel bir mesaj döndürür.
type MessageResponse struct {
	Message string `json:"message"`
}

// ErrorResponse, hata mesajını döndürür.
// Bu struct'ın zaten projenizde başka bir yerde tanımlı olduğunu varsayıyorum,
// eğer değilse aşağıdaki gibi tanımlanabilir:
// type ErrorResponse struct {
//	 Error string `json:"error"`
// }

type PermissionHandler struct {
	repo *repository.Repository
}

func NewPermissionHandler(repo *repository.Repository) *PermissionHandler {
	return &PermissionHandler{
		repo: repo,
	}
}

// @Tags Permissions
// @Summary Share a document with a user (send invitation)
// @Description Share a document with a user by providing the access type (viewer, editor). This creates a pending invitation.
// @Accept json
// @Produce json
// @Param id path string true "Document ID"
// @Param ShareDocumentRequest body ShareDocumentRequest true "Share document request (access_type: 'viewer' or 'editor')"
// @Success 201 {object} PermissionResponse "Invitation sent successfully"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /documents/{id}/permissions/share [post]
func (h *PermissionHandler) ShareDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz belge ID'si"})
		return
	}

	sharerUserID, err := utils.GetUserIDFromContext(c) // Daveti gönderen kullanıcı
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Belge bulunamadı"})
		return
	}

	// Yetkilendirme: Sadece doküman sahibi veya doküman üzerinde "admin" yetkisi olanlar paylaşabilir
	if doc.OwnerID != sharerUserID {
		// Burada GetByDocumentAndUser yerine GetAcceptedByDocumentAndUser kullanmak daha doğru olabilir
		// çünkü admin yetkisinin kabul edilmiş bir izin olması gerekir.
		permission, err := h.repo.Permission.GetAcceptedByDocumentAndUser(docID, sharerUserID)
		if err != nil || permission == nil || permission.AccessType != "admin" {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgeyi paylaşma izniniz yok"})
			return
		}
	}

	var shareRequest ShareDocumentRequest
	if err := c.ShouldBindJSON(&shareRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz istek formatı: " + err.Error()})
		return
	}

	if shareRequest.AccessType != string(models.AccessTypeViewer) && shareRequest.AccessType != string(models.AccessTypeEditor) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz erişim tipi. 'viewer' veya 'editor' olmalıdır"})
		return
	}

	targetUser, err := h.repo.User.GetByEmail(shareRequest.UserEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Davet edilecek kullanıcı bulunamadı"})
		return
	}

	if targetUser.ID == sharerUserID {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Belgeyi kendinizle paylaşamazsınız"})
		return
	}

	// Mevcut izni/davetiyeyi kontrol et
	existingPermission, err := h.repo.Permission.GetByDocumentAndUserWithAnyStatus(docID, targetUser.ID)
	if err == nil && existingPermission != nil {
		// Eğer zaten kabul edilmiş bir izin varsa ve yetki tipi farklıysa güncelle (opsiyonel, isteğe bağlı)
		if existingPermission.Status == models.PermissionStatusAccepted {
			if existingPermission.AccessType != shareRequest.AccessType {
				if err := h.repo.Permission.UpdateAccessType(existingPermission.ID, shareRequest.AccessType); err != nil {
					c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Mevcut erişim izni güncellenemedi: " + err.Error()})
					return
				}
				c.JSON(http.StatusOK, MessageResponse{Message: "Kullanıcının erişim izni güncellendi."})
				return
			}
			c.JSON(http.StatusOK, MessageResponse{Message: "Kullanıcının zaten bu erişim tipinde bir izni var."})
			return
		}
		// Eğer bekleyen bir davetiye varsa ve yetki tipi farklıysa güncelle
		if existingPermission.Status == models.PermissionStatusPending {
			if existingPermission.AccessType != shareRequest.AccessType {
				// GORM'da Update ile birden fazla alanı güncellemek için map veya struct kullanmak daha iyi.
				// Şimdilik ayrı metodlar varsayımıyla devam ediyorum, UpdateAccessType sadece access_type'ı güncelliyor.
				// Eğer Status'ü de güncellemek gerekiyorsa, ya UpdateStatus metodunu çağırın ya da
				// PermissionRepository'e yeni bir Update metod ekleyin.
				// Burada sadece access_type güncelleniyor, status pending kalıyor.
				if err := h.repo.Permission.UpdateAccessType(existingPermission.ID, shareRequest.AccessType); err != nil { // UpdateAccessType'ın status'ü değiştirmemesi lazım
					c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Bekleyen davetiyenin erişim tipi güncellenemedi: " + err.Error()})
					return
				}
				// shared_by ve updated_at de güncellenebilir.
				// _ = h.repo.Permission.UpdateSharedBy(existingPermission.ID, sharerUserID) // Örnek
				c.JSON(http.StatusOK, MessageResponse{Message: "Bekleyen davetiyenin erişim tipi güncellendi."})
				return
			}
			c.JSON(http.StatusOK, MessageResponse{Message: "Bu kullanıcı için zaten bekleyen bir davetiye var."})
			return
		}
		// Eğer reddedilmiş bir davetiye varsa, yeni bir davetiye oluşturulabilir (aşağıdaki kodla devam edecek).
	}

	newPermission := &models.Permission{
		DocumentID: docID,
		UserID:     targetUser.ID,
		AccessType: shareRequest.AccessType,
		Status:     models.PermissionStatusPending, // Durumu 'pending' olarak ayarla
		SharedBy:   sharerUserID,                   // Daveti göndereni kaydet
	}

	if err := h.repo.Permission.Create(newPermission); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Davetiye oluşturulamadı: " + err.Error()})
		return
	}

	var sharerUser models.User
	err = h.repo.User.GetByID(sharerUserID, &sharerUser)

	response := PermissionResponse{
		ID:         newPermission.ID,
		DocumentID: newPermission.DocumentID,
		UserID:     newPermission.UserID,
		UserEmail:  targetUser.Email,
		AccessType: newPermission.AccessType,
		Status:     newPermission.Status,
	}

	// sharerUser alınırken bir hata oluşmadıysa e-postasını ekle
	if err == nil { // Hata kontrolü
		response.SharedBy = sharerUser.Email
	} else {
		log.Printf("ShareDocument: Daveti gönderen kullanıcı bilgisi alınamadı (ID: %s), hata: %v", sharerUserID, err)
		// response.SharedBy boş kalacak veya bir varsayılan değer atanabilir.
	}

	c.JSON(http.StatusCreated, response)
}

// RemoveAccess bir kullanıcının dokümana olan kabul edilmiş erişimini veya bekleyen davetiyesini kaldırır/reddeder.
// @Tags Permissions
// @Summary Remove a user's access to a document or reject/cancel an invitation
// @Description Remove a specific user's accepted access to a document, or reject/cancel a pending invitation.
// @Accept json
// @Produce json
// @Param id path string true "Document ID"
// @Param RemoveAccessRequest body RemoveAccessRequest true "Remove access request"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /documents/{id}/permissions/remove [post] // Swagger için route güncellendi
func (h *PermissionHandler) RemoveAccess(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz belge ID'si"})
		return
	}

	removerUserID, err := utils.GetUserIDFromContext(c) // İşlemi yapan kullanıcı
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Belge bulunamadı"})
		return
	}

	var removeRequest RemoveAccessRequest
	if err := c.ShouldBindJSON(&removeRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz istek formatı"})
		return
	}

	targetUser, err := h.repo.User.GetByEmail(removeRequest.UserEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Erişimi kaldırılacak kullanıcı bulunamadı"})
		return
	}

	// Erişimi kaldırılacak izni/davetiyeyi bul
	permissionToRemove, err := h.repo.Permission.GetByDocumentAndUserWithAnyStatus(docID, targetUser.ID)
	if err != nil || permissionToRemove == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Kullanıcının bu doküman için bir izni veya davetiyesi bulunamadı"})
		return
	}

	// Yetkilendirme: Sadece doküman sahibi veya doküman üzerinde "admin" yetkisi olanlar erişimi kaldırabilir.
	// VEYA, eğer bekleyen bir davetiyeyse ve daveti gönderen kişi işlemi yapıyorsa.
	isOwner := doc.OwnerID == removerUserID
	isAdmin := false
	if !isOwner {
		removerPermission, _ := h.repo.Permission.GetAcceptedByDocumentAndUser(docID, removerUserID)
		if removerPermission != nil && removerPermission.AccessType == "admin" {
			isAdmin = true
		}
	}
	isSelfCancellingPending := permissionToRemove.Status == models.PermissionStatusPending && permissionToRemove.SharedBy == removerUserID && targetUser.ID != removerUserID
	// Kendi bekleyen davetiyesini de hedef kullanıcı reddedebilir (bu Accept/Reject endpoint'lerinde ele alınacak)
	// isTargetUserRejectingOwnPending := permissionToRemove.Status == models.PermissionStatusPending && permissionToRemove.UserID == removerUserID

	if !isOwner && !isAdmin && !isSelfCancellingPending {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgenin erişimini kaldırma izniniz yok"})
		return
	}

	// Eğer "pending" bir davetiyeyse ve admin/owner kaldırıyorsa, direkt silinebilir veya "rejected" yapılabilir.
	// Eğer "accepted" bir izni kaldırıyorsa, direkt silinebilir.
	// Bu örnekte direkt siliyoruz. DeleteByDocumentAndUser metodunun tüm statüleri silebildiğini varsayıyoruz.
	if err := h.repo.Permission.DeleteByDocumentAndUser(docID, targetUser.ID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Erişim kaldırılamadı: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Kullanıcının doküman erişimi/davetiyesi başarıyla kaldırıldı"})
}

// GetDocumentPermissions bir dokümanın KABUL EDİLMİŞ tüm izinlerini listeler.
// @Tags Permissions
// @Summary Get accepted permissions of a document
// @Description Get all accepted permissions for a specific document
// @Accept json
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {array} PermissionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /documents/{id}/permissions [get] // Swagger için route güncellendi
func (h *PermissionHandler) GetDocumentPermissions(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz belge ID'si"})
		return
	}

	requestingUserID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Belge bulunamadı"})
		return
	}

	// Yetkilendirme: Sadece doküman sahibi veya doküman üzerinde "admin" yetkisi olanlar izinleri görebilir.
	if doc.OwnerID != requestingUserID {
		permission, err := h.repo.Permission.GetAcceptedByDocumentAndUser(docID, requestingUserID)
		if err != nil || permission == nil || permission.AccessType != "admin" {
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu belgenin izinlerini görüntüleme yetkiniz yok"})
			return
		}
	}

	// GetByDocument artık sadece kabul edilmişleri getirmeli (repository'de güncellendi varsayımı)
	permissions, err := h.repo.Permission.GetByDocument(docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "İzinler alınamadı: " + err.Error()})
		return
	}

	var permissionResponses []PermissionResponse
	for _, perm := range permissions {
		user := models.User{}
		if err := h.repo.User.GetByID(perm.UserID, &user); err != nil {
			log.Printf("GetDocumentPermissions: Kullanıcı bulunamadı (ID: %s), atlanıyor. Hata: %v", perm.UserID, err)
			continue // Kullanıcı bulunamazsa bu izni atla
		}
		sharerEmail := ""
		if perm.SharedBy != uuid.Nil {
			sharer := models.User{}
			if err := h.repo.User.GetByID(perm.SharedBy, &sharer); err == nil {
				sharerEmail = sharer.Email
			}
		}

		permissionResponses = append(permissionResponses, PermissionResponse{
			ID:         perm.ID,
			DocumentID: perm.DocumentID,
			UserID:     perm.UserID,
			UserEmail:  user.Email,
			AccessType: perm.AccessType,
			Status:     perm.Status,
			SharedBy:   sharerEmail,
		})
	}

	c.JSON(http.StatusOK, permissionResponses)
}

// --- Yeni Handler'lar ---

// @Tags Invitations
// @Summary Get pending invitations for the current user
// @Description Retrieves all pending document share invitations for the authenticated user.
// @Produce json
// @Success 200 {array} InvitationDetailResponse
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /invitations/pending [get]
func (h *PermissionHandler) GetPendingInvitations(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	invitations, err := h.repo.Permission.GetPendingInvitationsByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Bekleyen davetiyeler alınamadı: " + err.Error()})
		return
	}

	var responses []InvitationDetailResponse
	for _, inv := range invitations {
		doc := models.Document{}
		docTitle := "Bilinmeyen Belge"
		if err := h.repo.Document.GetByID(inv.DocumentID, &doc); err == nil {
			docTitle = doc.Title
		}

		sharer := models.User{}
		sharerEmail := "Bilinmeyen Paylaşan"
		if err := h.repo.User.GetByID(inv.SharedBy, &sharer); err == nil {
			sharerEmail = sharer.Email
		}

		responses = append(responses, InvitationDetailResponse{
			InvitationID:  inv.ID,
			DocumentID:    inv.DocumentID,
			DocumentTitle: docTitle,
			SharedByEmail: sharerEmail,
			AccessType:    inv.AccessType,
			Status:        inv.Status,
			CreatedAt:     inv.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, responses)
}

// @Tags Invitations
// @Summary Accept a pending invitation
// @Description Accepts a pending document share invitation.
// @Param invitation_id path string true "Invitation ID (Permission ID)"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse "Invalid invitation ID"
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 403 {object} ErrorResponse "Forbidden (not your invitation or not pending)"
// @Failure 404 {object} ErrorResponse "Invitation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /invitations/{invitation_id}/accept [post]
func (h *PermissionHandler) AcceptInvitation(c *gin.Context) {
	invitationIDStr := c.Param("invitation_id")
	invitationID, err := uuid.Parse(invitationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz davetiye ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var invitation models.Permission
	if err := h.repo.Permission.GetByID(invitationID, &invitation); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Davetiye bulunamadı"})
		return
	}

	if invitation.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu davetiyeyi kabul etme yetkiniz yok"})
		return
	}

	if invitation.Status != models.PermissionStatusPending {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu davetiye zaten yanıtlanmış veya geçersiz."})
		return
	}

	if err := h.repo.Permission.UpdateStatus(invitationID, models.PermissionStatusAccepted); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Davetiye kabul edilemedi: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Davetiye başarıyla kabul edildi."})
}

// @Tags Invitations
// @Summary Reject a pending invitation
// @Description Rejects a pending document share invitation.
// @Param invitation_id path string true "Invitation ID (Permission ID)"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse "Invalid invitation ID"
// @Failure 401 {object} ErrorResponse "Authentication error"
// @Failure 403 {object} ErrorResponse "Forbidden (not your invitation or not pending)"
// @Failure 404 {object} ErrorResponse "Invitation not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /invitations/{invitation_id}/reject [post]
func (h *PermissionHandler) RejectInvitation(c *gin.Context) {
	invitationIDStr := c.Param("invitation_id")
	invitationID, err := uuid.Parse(invitationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz davetiye ID'si"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Kimlik doğrulama hatası"})
		return
	}

	var invitation models.Permission
	if err := h.repo.Permission.GetByID(invitationID, &invitation); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Davetiye bulunamadı"})
		return
	}

	if invitation.UserID != userID {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu davetiyeyi reddetme yetkiniz yok"})
		return
	}

	if invitation.Status != models.PermissionStatusPending {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Bu davetiye zaten yanıtlanmış veya geçersiz."})
		return
	}
	// İsteğe bağlı: Reddedilen davetiyeyi silmek yerine sadece status'ü güncellemek daha iyi olabilir.
	// Bu örnekte status'ü güncelliyoruz.
	if err := h.repo.Permission.UpdateStatus(invitationID, models.PermissionStatusRejected); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Davetiye reddedilemedi: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{Message: "Davetiye başarıyla reddedildi."})
}

type GetUserDocumentPermissionRequest struct {
	DocumentId string `json:"document_id" binding:"required"`
}

type GetUserDocumentPermissionResponse struct {
	AccessType string `json:"access_type"`
}

// GetDocumentPermission godoc
// @Summary      Get user's permission for a document
// @Description  Retrieves the access type for the authenticated user on a specific document.
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Param        permissionRequest body GetDocumentPermissionRequest true "Document ID"
// @Success      200  {object}  GetDocumentPermissionResponse
// @Failure      400  {object}  ErrorResponse "Invalid request format or invalid UUID"
// @Failure      401  {object}  ErrorResponse "Unauthorized"
// @Failure      404  {object}  ErrorResponse "Permission not found for this document"
// @Failure      500  {object}  ErrorResponse "Internal server error"
// @Router       /api/v1/permissions/document [post] // Örnek bir rota, kendi rotanızla güncelleyin
func (h *PermissionHandler) GetUserDocumentPermission(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	var permissionRequest GetUserDocumentPermissionRequest
	if err := c.ShouldBindJSON(&permissionRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz istek formatı: " + err.Error()})
		return
	}

	documentUUID, err := uuid.Parse(permissionRequest.DocumentId)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz doküman ID formatı."})
		return
	}

	permission, err := h.repo.Permission.GetByDocumentAndUser(documentUUID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Bu doküman için yetkiniz bulunmamaktadır."})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Yetki kontrolü sırasında bir hata oluştu."})
		return
	}

	c.JSON(http.StatusOK, GetUserDocumentPermissionResponse{
		AccessType: permission.AccessType,
	})
}
