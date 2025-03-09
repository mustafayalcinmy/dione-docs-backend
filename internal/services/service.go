package services

import "github.com/dione-docs-backend/internal/repository"

type Service struct {
}

func NewService(repo *repository.Repository) *Service {
	return &Service{}
}
