package models

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID `gorm:"type:uuid;index;not null"`
	DocumentID  uuid.UUID `gorm:"type:uuid;index;not null"`
	AccessLevel string    `gorm:"type:text;check:access_level IN ('view','edit');not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	User        User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Document    Document `gorm:"foreignKey:DocumentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (Permission) TableName() string {
	return "permissions"
}

func (Permission) TableUnique() []interface{} {
	return []interface{}{
		"user_id",
		"document_id",
	}
}
