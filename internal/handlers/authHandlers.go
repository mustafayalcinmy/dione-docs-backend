package handlers

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafayalcinmy/dione-docs-backend/internal/auth"
	"github.com/mustafayalcinmy/dione-docs-backend/internal/database"
	"golang.org/x/crypto/bcrypt"
)

func (apiCfg *ApiConfig) GetLoginPage(c *gin.Context) {
	tmpl, err := template.ParseFiles(
		"internal/presentation/templates/auth/login.html",
		"internal/presentation/templates/partials/navbar.html",
		"internal/presentation/templates/base.html",
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error parsing templates")
		log.Printf("Error parsing templates: %v", err)
		return
	}

	data := struct {
		Title  string
		Navbar bool
	}{
		Title:  "Login",
		Navbar: false,
	}

	err = tmpl.ExecuteTemplate(c.Writer, "base.html", data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error executing template")
		log.Printf("Error executing template: %v", err)
	}
}

func (apiCfg *ApiConfig) GetRegisterPage(c *gin.Context) {
	tmpl, err := template.ParseFiles(
		"internal/presentation/templates/auth/register.html",
		"internal/presentation/templates/partials/navbar.html",
		"internal/presentation/templates/base.html",
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error parsing templates")
		log.Printf("Error parsing templates: %v", err)
		return
	}

	data := struct {
		Title  string
		Navbar bool
	}{
		Title:  "Register",
		Navbar: false,
	}

	err = tmpl.ExecuteTemplate(c.Writer, "base.html", data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error executing template")
		log.Printf("Error executing template: %v", err)
	}
}

func (apiCfg *ApiConfig) PostLogin(c *gin.Context) {
	type loginParams struct {
		Username string `form:"username" json:"username"`
		Password string `form:"password" json:"password"`
	}

	var params loginParams
	if err := c.ShouldBind(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	user, err := apiCfg.DB.GetUserByUsername(c, params.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(params.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	token, err := auth.GenerateJWT(params.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	expirationTime := time.Now().Add(30 * time.Minute)
	c.SetCookie("token", token, int(expirationTime.Sub(time.Now()).Seconds()), "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/")
}

func (apiCfg *ApiConfig) PostLogout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.Redirect(http.StatusSeeOther, "/")
}

func (apiCfg *ApiConfig) PostRegister(c *gin.Context) {
	var req struct {
		Fullname string `form:"fullname" binding:"required"`
		Username string `form:"username" binding:"required"`
		Email    string `form:"email" binding:"required,email"`
		Password string `form:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	user := database.User{
		ID:           uuid.New(),
		Fullname:     req.Fullname,
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	user, err = apiCfg.DB.CreateUser(c, database.CreateUserParams{
		Username:     user.Username,
		Fullname:     user.Fullname,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "e": err})
		return
	}

	token, err := auth.GenerateJWT(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	expirationTime := time.Now().Add(30 * time.Minute)
	c.SetCookie("token", token, int(expirationTime.Sub(time.Now()).Seconds()), "/", "", false, true)

	c.Redirect(http.StatusSeeOther, "/")
}
