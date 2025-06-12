package api

import (
	"github.com/dione-docs-backend/internal/api/handlers"
	middleware "github.com/dione-docs-backend/internal/api/middlewares"
	"github.com/dione-docs-backend/internal/config"
	"github.com/dione-docs-backend/internal/parser/docx"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/dione-docs-backend/internal/services"
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
	// Instantiate Parsers
	docxParser := docx.NewManualParser()
	// Instantiate Services
	importService := services.NewImportService(r.repository.Document, docxParser)

	// Instantiate Handlers
	authHandler := handlers.NewAuthHandler(r.repository, r.config)
	docHandler := handlers.NewDocumentHandler(r.repository)
	permHandler := handlers.NewPermissionHandler(r.repository)
	importHandler := handlers.NewImportHandler(importService)

	otHubManager := handlers.NewHubManager(r.repository.Document)

	chatHubManager := handlers.NewChatHubManager(r.repository)
	chatHandler := handlers.NewChatHandler(r.repository, chatHubManager, r.config)

	// Public routes
	apiPublic := r.engine.Group("/api/v1")
	{
		apiPublic.POST("/register", authHandler.RegisterHandler)
		apiPublic.POST("/login", authHandler.LoginHandler)
	}

	// Authenticated routes
	apiAuth := r.engine.Group("/api/v1")
	apiAuth.Use(middleware.JWTMiddleware(r.config))
	{
		apiAuth.GET("/me", authHandler.GetCurrentUser)

		apiAuth.GET("/ws/documents/:id", otHubManager.ServeWs)

		apiAuth.GET("/ws/chat/documents/:id", chatHandler.ServeChatWs)

		// Document Routes
		docs := apiAuth.Group("/documents")
		{
			docs.POST("", docHandler.CreateDocument)
			docs.GET("/user", docHandler.GetUserDocuments)
			docs.GET("/:id", docHandler.GetDocument)
			docs.PUT("/:id", docHandler.UpdateDocument)
			docs.DELETE("/:id", docHandler.DeleteDocument)
			docs.GET("/:id/versions", docHandler.GetDocumentVersions)

			// YENİ: Chat geçmişini getirmek için REST endpoint'i
			docs.GET("/:id/messages", chatHandler.GetMessages)

			docs.POST("/:id/permissions/share", permHandler.ShareDocument)
			docs.POST("/:id/permissions/remove", permHandler.RemoveAccess)
			docs.GET("/:id/permissions", permHandler.GetDocumentPermissions)
			docs.GET("/:id/role", permHandler.GetUserDocumentPermission)
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

	apiInternal := r.engine.Group("/api/v1/internal")
	apiInternal.Use(middleware.InternalAPIMiddleware(r.config))
	{
		docsInternal := apiInternal.Group("/documents")
		{
			docsInternal.PUT("/:id/content", docHandler.UpdateDocumentContent)
		}
	}

	// Swagger documentation route
	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
