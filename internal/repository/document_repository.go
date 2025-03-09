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
	GetByOwner(ownerID uuid.UUID) ([]*models.Document, error)
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

func (r *documentRepo) GetByOwner(ownerID uuid.UUID) ([]*models.Document, error) {
	var docs []*models.Document
	if err := r.db.Where("created_by = ?", ownerID).Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}
