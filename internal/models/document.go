package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Document struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Title       string    `gorm:"not null"`
	Description string
	OwnerID     uuid.UUID `gorm:"type:uuid;not null"`
	Content     []byte    `gorm:"type:jsonb"`
	Version     int       `gorm:"not null;default:1"`
	IsPublic    bool      `gorm:"not null;default:false"`
	Status      string    `gorm:"not null;default:'draft'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type DocumentVersion struct {
	ID         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null"`
	Version    int       `gorm:"not null"`
	Content    []byte    `gorm:"type:jsonb"`
	ChangedBy  uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt  time.Time
}
