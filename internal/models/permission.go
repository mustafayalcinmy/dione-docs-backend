package models

import (
	"time"

	"github.com/google/uuid"
)

type PermissionStatus string

const (
	PermissionStatusPending  PermissionStatus = "pending"
	PermissionStatusAccepted PermissionStatus = "accepted"
	PermissionStatusRejected PermissionStatus = "rejected"
)

type AccessType string

const (
	AccessTypeViewer AccessType = "viewer"
	AccessTypeEditor AccessType = "editor"
	AccessTypeAdmin  AccessType = "admin"
)

type Permission struct {
	ID         uuid.UUID        `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	DocumentID uuid.UUID        `gorm:"type:uuid;not null;index:idx_doc_user_status,unique"`
	UserID     uuid.UUID        `gorm:"type:uuid;not null;index:idx_doc_user_status,unique"`
	AccessType string           `gorm:"not null"`
	Status     PermissionStatus `gorm:"type:varchar(10);not null;default:'pending';index:idx_doc_user_status,unique"`
	SharedBy   uuid.UUID        `gorm:"type:uuid;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
