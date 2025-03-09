package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Document struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Title     string    `gorm:"not null"`
	Content   string    `gorm:"type:text"`
	CreatedBy uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Owner       User         `gorm:"foreignKey:CreatedBy;references:ID"`
	Permissions []Permission `gorm:"foreignKey:DocumentID"`
}
