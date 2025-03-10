package models

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null"`
	UserID     uuid.UUID `gorm:"type:uuid;not null"`
	AccessType string    `gorm:"not null"` // read, edit, admin
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
