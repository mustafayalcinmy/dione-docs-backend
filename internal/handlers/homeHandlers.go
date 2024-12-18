package handlers

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (apiCfg *ApiConfig) HomeHandler(c *gin.Context) {
	username, exists := c.Get("username")
	var usernameStr string
	if exists {
		usernameStr, _ = username.(string)
	}

	data := struct {
		Title    string
		Navbar   bool
		Username string
	}{
		Title:    "Homepage",
		Navbar:   true,
		Username: usernameStr,
	}

	tmpl := template.Must(template.ParseFiles(
		"internal/presentation/templates/base.html",
		"internal/presentation/templates/partials/navbar.html",
		"internal/presentation/templates/home.html",
	))
	c.Status(http.StatusOK)
	tmpl.ExecuteTemplate(c.Writer, "base.html", data)
}
