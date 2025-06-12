package repository

import (
	"github.com/dione-docs-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(message *models.Message) error
	GetByDocumentID(documentID uuid.UUID) ([]models.Message, error)
	GetByID(id uuid.UUID) (*models.Message, error)
}
type messageRepo struct {
	*GenericRepository[models.Message]
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepo{
		GenericRepository: NewGenericRepository[models.Message](db),
		db:                db,
	}
}

func (r *messageRepo) GetByDocumentID(documentID uuid.UUID) ([]models.Message, error) {
	var messages []models.Message
	if err := r.db.Preload("User").Where("document_id = ?", documentID).Order("created_at asc").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *messageRepo) GetByID(id uuid.UUID) (*models.Message, error) {
	var message models.Message
	if err := r.db.Preload("User").First(&message, id).Error; err != nil {
		return nil, err
	}
	return &message, nil
}
