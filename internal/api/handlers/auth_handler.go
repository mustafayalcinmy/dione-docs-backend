package handlers

import (
	"net/http"
	"time"

	"github.com/dione-docs-backend/internal/config"
	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func LoginHandler(repo repository.UserRepository, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request data"})
			return
		}

		user, err := repo.GetByEmail(req.Email)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid credentials"})
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": user.ID.String(),
			"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
		})
		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, LoginResponse{Token: tokenString})
	}
}

func RegisterHandler(repo repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request data"})
			return
		}

		_, err := repo.GetByEmail(req.Email)
		if err == nil {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "Email already in use"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to hash password"})
			return
		}

		user := &models.User{
			ID:           uuid.New(),
			Username:     req.Username,
			Email:        req.Email,
			PasswordHash: string(hashedPassword),
		}

		if err := repo.Create(user); err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create user"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
	}
}
