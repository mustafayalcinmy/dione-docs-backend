package repository

import (
	"github.com/dione-docs-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermissionRepository interface {
	Create(permission *models.Permission) error
	GetByID(id any, permission *models.Permission) error
	GetByDocumentAndUser(documentID, userID uuid.UUID) (*models.Permission, error)
	GetByDocument(documentID uuid.UUID) ([]models.Permission, error)                       // Bu metod sadece kabul edilmişleri mi getirmeli? Yoksa tümünü mü? Şimdilik tümünü getiriyor.
	GetAcceptedByDocumentAndUser(documentID, userID uuid.UUID) (*models.Permission, error) // Sadece kabul edilmiş izni getirir
	UpdateAccessType(permissionID uuid.UUID, accessType string) error
	DeleteByDocumentAndUser(documentID, userID uuid.UUID) error // Bu metod, davetiye sistemiyle nasıl çalışacak? Belki direkt silmek yerine status'u rejected yapmak daha iyi. Şimdilik koruyorum.

	// --- Yeni Metodlar ---
	GetPendingInvitationsByUserID(userID uuid.UUID) ([]models.Permission, error)
	UpdateStatus(permissionID uuid.UUID, status models.PermissionStatus) error
	GetByDocumentAndUserWithAnyStatus(documentID, userID uuid.UUID) (*models.Permission, error) // Herhangi bir statüdeki izni/davetiyeyi getirir
}

type permissionRepo struct {
	*GenericRepository[models.Permission]
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepo{
		GenericRepository: NewGenericRepository[models.Permission](db),
		db:                db,
	}
}

// Create metodu artık status'ü dikkate alarak çalışacak,
// ShareDocument handler'ında permission objesi oluşturulurken Status="pending" ve SharedBy atanmalı.
// GenericRepository.Create kullanıldığı için burada özel bir Create implementasyonuna gerek yok,
// modeldeki default değer veya handler'daki atama yeterli.

// GetByDocumentAndUser metodu, bir kullanıcının bir dokümanda AKTİF (kabul edilmiş) bir izni olup olmadığını kontrol eder.
// Orijinal fonksiyonu koruyarak, sadece kabul edilmiş izinleri döndürecek şekilde güncelliyoruz.
// Eğer herhangi bir statüdeki izni getirmek isterseniz GetByDocumentAndUserWithAnyStatus metodunu kullanın.
func (r *permissionRepo) GetByDocumentAndUser(documentID, userID uuid.UUID) (*models.Permission, error) {
	var permission models.Permission
	if err := r.db.Where("document_id = ? AND user_id = ? AND status = ?", documentID, userID, models.PermissionStatusAccepted).
		First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetAcceptedByDocumentAndUser sadece kabul edilmiş ve belirli bir accessType'a sahip izni getirir.
// Middleware veya yetkilendirme kontrolleri için kullanılabilir.
func (r *permissionRepo) GetAcceptedByDocumentAndUser(documentID, userID uuid.UUID) (*models.Permission, error) {
	var permission models.Permission
	if err := r.db.Where("document_id = ? AND user_id = ? AND status = ?", documentID, userID, models.PermissionStatusAccepted).
		First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetByDocumentAndUserWithAnyStatus metodu, bir kullanıcının bir doküman için herhangi bir statüde (pending, accepted, rejected)
// bir izni veya davetiyesi olup olmadığını kontrol etmek için kullanılır.
func (r *permissionRepo) GetByDocumentAndUserWithAnyStatus(documentID, userID uuid.UUID) (*models.Permission, error) {
	var permission models.Permission
	if err := r.db.Where("document_id = ? AND user_id = ?", documentID, userID).
		Order("created_at desc"). // En son kaydı almak için
		First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetByDocument metodu, bir dokümana ait tüm KABUL EDİLMİŞ izinleri listeler.
// Eğer tüm statüleri listelemek gerekirse yeni bir metod eklenebilir.
func (r *permissionRepo) GetByDocument(documentID uuid.UUID) ([]models.Permission, error) {
	var permissions []models.Permission
	if err := r.db.Where("document_id = ? AND status = ?", documentID, models.PermissionStatusAccepted).
		Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *permissionRepo) UpdateAccessType(permissionID uuid.UUID, accessType string) error {
	return r.db.Model(&models.Permission{}).
		Where("id = ? AND status = ?", permissionID, models.PermissionStatusAccepted). // Sadece kabul edilmişlerin yetkisi değişebilir
		Update("access_type", accessType).Error
}

// DeleteByDocumentAndUser metodu bir izni/davetiyeyi direkt siler.
// Bunun yerine UpdateStatus ile 'rejected' yapmak daha iyi bir pratik olabilir,
// böylece kullanıcıya tekrar davet gönderildiğinde veya geçmişi görmek istediğimizde sorun yaşamayız.
// Şimdilik bu metodun işlevini koruyorum, ihtiyaca göre değiştirilebilir.
func (r *permissionRepo) DeleteByDocumentAndUser(documentID, userID uuid.UUID) error {
	return r.db.Where("document_id = ? AND user_id = ?", documentID, userID).
		Delete(&models.Permission{}).Error
}

// --- Yeni Metodlar Implementasyonu ---

func (r *permissionRepo) GetPendingInvitationsByUserID(userID uuid.UUID) ([]models.Permission, error) {
	var invitations []models.Permission
	if err := r.db.Where("user_id = ? AND status = ?", userID, models.PermissionStatusPending).
		Order("created_at desc").
		Find(&invitations).Error; err != nil {
		return nil, err
	}
	return invitations, nil
}

func (r *permissionRepo) UpdateStatus(permissionID uuid.UUID, status models.PermissionStatus) error {
	return r.db.Model(&models.Permission{}).
		Where("id = ?", permissionID).
		Update("status", status).Error
}
