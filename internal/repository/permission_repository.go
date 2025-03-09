package repository

import (
	"github.com/dione-docs-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermissionRepository interface {
	Create(perm *models.Permission) error
	Delete(perm *models.Permission) error
	GetByID(id any, perm *models.Permission) error
	GetByUserID(userID uuid.UUID) ([]*models.Permission, error)
	GetByDocumentID(docID uuid.UUID) ([]*models.Permission, error)
	GetByUserAndDocument(userID uuid.UUID, docID uuid.UUID) (*models.Permission, error)
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

func (r *permissionRepo) GetByUserID(userID uuid.UUID) ([]*models.Permission, error) {
	var perms []*models.Permission
	if err := r.db.Where("user_id = ?", userID).Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *permissionRepo) GetByDocumentID(docID uuid.UUID) ([]*models.Permission, error) {
	var perms []*models.Permission
	if err := r.db.Where("document_id = ?", docID).Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *permissionRepo) GetByUserAndDocument(userID uuid.UUID, docID uuid.UUID) (*models.Permission, error) {
	var perm models.Permission
	if err := r.db.Where("user_id = ? AND document_id = ?", userID, docID).First(&perm).Error; err != nil {
		return nil, err
	}
	return &perm, nil
}
