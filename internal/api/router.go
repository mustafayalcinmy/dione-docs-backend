package api

import (
	"github.com/dione-docs-backend/internal/api/handlers"
	middleware "github.com/dione-docs-backend/internal/api/middlewares" // Alias if needed
	"github.com/dione-docs-backend/internal/config"
	"github.com/dione-docs-backend/internal/parser/docx" // Import the docx parser package
	"github.com/dione-docs-backend/internal/repository"
	"github.com/dione-docs-backend/internal/services" // Import services package
	"github.com/gin-gonic/gin"

	_ "github.com/dione-docs-backend/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Router struct {
	engine     *gin.Engine
	repository *repository.Repository
	config     *config.Config
	// Add services if needed for setup
}

func NewRouter(repo *repository.Repository, cfg *config.Config) *Router {
	r := &Router{
		engine:     gin.New(),
		repository: repo,
		config:     cfg,
	}
	r.setupMiddlewares()
	r.setupRoutes() // Pass repo and cfg to setupRoutes if handlers need them directly
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
		// TODO: Apply JWTMiddleware to protected routes like import
	)
}

func (r *Router) setupRoutes() {
	// Instantiate Parsers
	docxParser := docx.NewManualParser()
	// Instantiate Services
	importService := services.NewImportService(r.repository.Document, docxParser) // Pass parser

	// Instantiate Handlers
	authHandler := handlers.NewAuthHandler(r.repository, r.config)
	docHandler := handlers.NewDocumentHandler(r.repository)
	permHandler := handlers.NewPermissionHandler(r.repository)
	importHandler := handlers.NewImportHandler(importService) // New handler

	// Public routes
	apiPublic := r.engine.Group("/api/v1")
	{
		apiPublic.POST("/register", authHandler.RegisterHandler)
		apiPublic.POST("/login", authHandler.LoginHandler)
	}

	// Authenticated routes
	apiAuth := r.engine.Group("/api/v1")
	// Apply JWT middleware to this group
	apiAuth.Use(middleware.JWTMiddleware(r.config)) // Make sure JWTMiddleware exists and is configured
	{

		apiAuth.GET("/me", authHandler.GetCurrentUser)
		// Document Routes
		docs := apiAuth.Group("/documents")
		{
			docs.POST("", docHandler.CreateDocument)
			docs.GET("/user", docHandler.GetUserDocuments)
			docs.GET("/:id", docHandler.GetDocument)

			docs.POST("/:id/permissions/share", permHandler.ShareDocument)
			docs.POST("/:id/permissions/remove", permHandler.RemoveAccess)
			docs.GET("/:id/permissions", permHandler.GetDocumentPermissions)

			docs.PUT("/:id", docHandler.UpdateDocument)
			docs.DELETE("/:id", docHandler.DeleteDocument)
			docs.GET("/:id/versions", docHandler.GetDocumentVersions)
		}

		invitations := apiAuth.Group("/invitations")
		{
			invitations.GET("/pending", permHandler.GetPendingInvitations)
			invitations.POST("/:invitation_id/accept", permHandler.AcceptInvitation)
			invitations.POST("/:invitation_id/reject", permHandler.RejectInvitation)
		}

		imp := apiAuth.Group("/import")
		{
			imp.POST("/docx", importHandler.ImportDocxHandler)
		}
	}

	// Swagger (usually public)
	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
