package repository

import (
	"github.com/dione-docs-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentRepository interface {
	Create(doc *models.Document) error
	Delete(doc *models.Document) error
	GetByID(id any, doc *models.Document) error
	Update(doc *models.Document) error
	GetByOwnerID(ownerID uuid.UUID) ([]models.Document, error)
	GetSharedWithUser(userID uuid.UUID) ([]models.Document, error)
	SaveVersion(version *models.DocumentVersion) error
	GetVersions(documentID uuid.UUID) ([]models.DocumentVersion, error)
}

type documentRepo struct {
	*GenericRepository[models.Document]
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) DocumentRepository {
	return &documentRepo{
		GenericRepository: NewGenericRepository[models.Document](db),
		db:                db,
	}
}

func (r *documentRepo) Update(doc *models.Document) error {
	return r.db.Save(doc).Error
}

func (r *documentRepo) GetByOwnerID(ownerID uuid.UUID) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.Where("owner_id = ?", ownerID).Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *documentRepo) GetSharedWithUser(userID uuid.UUID) ([]models.Document, error) {
	var docs []models.Document
	if err := r.db.Joins("JOIN permissions ON permissions.document_id = documents.id").
		Where("permissions.user_id = ?", userID).
		Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *documentRepo) SaveVersion(version *models.DocumentVersion) error {
	return r.db.Create(version).Error
}

func (r *documentRepo) GetVersions(documentID uuid.UUID) ([]models.DocumentVersion, error) {
	var versions []models.DocumentVersion
	if err := r.db.Where("document_id = ?", documentID).
		Order("version desc").
		Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *permissionRepo) GetByDocumentAndUser(documentID, userID uuid.UUID) (*models.Permission, error) {
	var permission models.Permission
	if err := r.db.Where("document_id = ? AND user_id = ?", documentID, userID).
		First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *permissionRepo) GetByDocument(documentID uuid.UUID) ([]models.Permission, error) {
	var permissions []models.Permission
	if err := r.db.Where("document_id = ?", documentID).
		Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *permissionRepo) UpdateAccessType(permissionID uuid.UUID, accessType string) error {
	return r.db.Model(&models.Permission{}).
		Where("id = ?", permissionID).
		Update("access_type", accessType).Error
}

func (r *permissionRepo) DeleteByDocumentAndUser(documentID, userID uuid.UUID) error {
	return r.db.Where("document_id = ? AND user_id = ?", documentID, userID).
		Delete(&models.Permission{}).Error
}
