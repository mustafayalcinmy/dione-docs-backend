package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafayalcinmy/dione-docs-backend/internal/database"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Username     string    `json:"username"`
	Fullname     string    `json:"fullname"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Hide password hash in JSON responses
}

func (u *User) FromDatabase(dbUser database.User) *User {
	return &User{
		ID:           dbUser.ID,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
		Username:     dbUser.Username,
		Fullname:     dbUser.Fullname,
		Email:        dbUser.Email,
		PasswordHash: dbUser.PasswordHash,
	}
}
