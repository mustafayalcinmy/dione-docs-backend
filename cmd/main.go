package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/mustafayalcinmy/dione-docs-backend/internal/auth"
	"github.com/mustafayalcinmy/dione-docs-backend/internal/database"
	"github.com/mustafayalcinmy/dione-docs-backend/internal/handlers"
)

func main() {
	godotenv.Load(".env")

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT is not found in the .env file!")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not found in the .env file!")
	}

	conn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Can't connect to the database")
	}

	apiCfg := handlers.ApiConfig{
		DB: database.New(conn),
	}

	router := gin.Default()

	staticDir := "./internal/presentation/assets"
	router.Static("/assets", staticDir)

	router.Use(auth.JWTMiddleware())

	webRouter := router.Group("")
	{
		webRouter.Use(auth.JWTMiddleware())
		{
			webRouter.GET("/", apiCfg.HomeHandler)
		}
		webRouter.GET("/login", apiCfg.GetLoginPage)
		webRouter.GET("/register", apiCfg.GetRegisterPage)
	}

	apiRouter := router.Group("/api")
	{
		apiRouter.POST("/login", apiCfg.PostLogin)
		apiRouter.POST("/logout", apiCfg.PostLogout)
		apiRouter.POST("/register", apiCfg.PostRegister)
	}

	adminRouter := router.Group("/admin")
	{
		adminRouter.GET("/users/create", apiCfg.GetLoginPage)
		adminRouter.GET("/users/edit/:id", apiCfg.GetLoginPage)
		adminRouter.GET("/users/details/:id", apiCfg.GetLoginPage)
	}

	fmt.Printf("Server starting on port: %s\n", port)
	err = router.Run(":" + port)
	if err != nil {
		log.Fatal(err)
	}
}
