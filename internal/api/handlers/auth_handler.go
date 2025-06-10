package handlers

import (
	"net/http"
	"time"

	"github.com/dione-docs-backend/internal/config"
	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/dione-docs-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewAuthHandler(repo *repository.Repository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		repo: repo,
		cfg:  cfg,
	}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type UserResponse struct {
	UserId   string `json:"userId"`
	Email    string `json:"email"`
	FullName string `json:"fullName"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type RegisterResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// LoginHandler godoc
// @Summary Login endpoint
// @Description Authenticates a user and returns a JWT token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param login body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 401 {object} ErrorResponse "Invalid credentials"
// @Router /api/v1/login [post]
func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request data"})
		return
	}

	user, err := h.repo.User.GetByEmail(req.Email)
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
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{Token: tokenString})
}

// RegisterHandler godoc
// @Summary Register endpoint
// @Description Registers a new user and returns a JWT token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param register body RegisterRequest true "Registration data"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse "Invalid request data"
// @Failure 409 {object} ErrorResponse "Email already in use"
// @Router /api/v1/register [post]
func (h *AuthHandler) RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request data"})
		return
	}

	_, err := h.repo.User.GetByEmail(req.Email)
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

	if err := h.repo.User.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create user"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{Token: tokenString})
}

// GetCurrentUser godoc
// @Summary Get current user info
// @Description Returns the current authenticated user's information
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserResponse
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Router /api/v1/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	// JWT middleware'den user_id'yi al
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Unauthorized"})
		return
	}

	// Kullanıcıyı veritabanından al
	var user models.User
	if err := h.repo.User.GetByID(userID, &user); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
		return
	}

	// Kullanıcı bilgilerini dön
	c.JSON(http.StatusOK, UserResponse{
		UserId:   user.ID.String(),
		Email:    user.Email,
		FullName: user.Username, // Eğer User model'inde FullName field'ı varsa onu kullan
	})
}
