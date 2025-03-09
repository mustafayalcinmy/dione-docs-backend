package repository

import "gorm.io/gorm"

type Repository struct {
	User       UserRepository
	Document   DocumentRepository
	Permission PermissionRepository
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		User:       NewUserRepository(db),
		Document:   NewDocumentRepository(db),
		Permission: NewPermissionRepository(db),
	}
}
