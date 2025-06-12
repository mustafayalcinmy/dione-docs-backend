package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null" json:"document_id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`

	UserName string `gorm:"not null" json:"-"`

	Content   string         `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user"`
}
