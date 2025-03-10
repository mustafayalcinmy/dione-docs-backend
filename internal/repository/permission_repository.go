package repository

import (
	"github.com/dione-docs-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermissionRepository interface {
	Create(permission *models.Permission) error
	Delete(permission *models.Permission) error
	GetByID(id any, permission *models.Permission) error
	GetByDocumentAndUser(documentID, userID uuid.UUID) (*models.Permission, error)
	GetByDocument(documentID uuid.UUID) ([]models.Permission, error)
	UpdateAccessType(permissionID uuid.UUID, accessType string) error
	DeleteByDocumentAndUser(documentID, userID uuid.UUID) error
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
