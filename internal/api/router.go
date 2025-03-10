package api

import (
	"github.com/dione-docs-backend/internal/api/handlers"
	middleware "github.com/dione-docs-backend/internal/api/middlewares"
	"github.com/dione-docs-backend/internal/config"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/gin-gonic/gin"

	_ "github.com/dione-docs-backend/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Router struct {
	engine     *gin.Engine
	repository *repository.Repository
	config     *config.Config
}

func NewRouter(repo *repository.Repository, cfg *config.Config) *Router {
	r := &Router{
		engine:     gin.New(),
		repository: repo,
		config:     cfg,
	}
	r.setupMiddlewares()
	r.setupRoutes()
	return r
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) setupMiddlewares() {
	r.engine.Use(
		gin.Logger(),
		gin.Recovery(),
		middleware.CORSMiddleware(),
	)
}

func (r *Router) setupRoutes() {
	authHandler := handlers.NewAuthHandler(r.repository, r.config)
	docHandler := handlers.NewDocumentHandler(r.repository)
	permHandler := handlers.NewPermissionHandler(r.repository)

	api := r.engine.Group("/api/v1")
	{
		api.POST("/register", authHandler.RegisterHandler)
		api.POST("/login", authHandler.LoginHandler)
	}

	docs := api.Group("/documents")
	{
		docs.POST("", docHandler.CreateDocument)
		docs.GET("", docHandler.GetUserDocuments)
		docs.GET("/:id", docHandler.GetDocument)
		docs.PUT("/:id", docHandler.UpdateDocument)
		docs.DELETE("/:id", docHandler.DeleteDocument)
		docs.GET("/:id/versions", docHandler.GetDocumentVersions)
	}

	perms := api.Group("/permissions")
	{
		perms.POST("/documents/:id/share", permHandler.ShareDocument)
		perms.DELETE("/documents/:id/access", permHandler.RemoveAccess)
		perms.GET("/documents/:id", permHandler.GetDocumentPermissions)
	}

	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
