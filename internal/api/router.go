// internal/api/router.go
package api

import (
	"github.com/dione-docs-backend/internal/api/handlers"
	middleware "github.com/dione-docs-backend/internal/api/middlewares"
	"github.com/dione-docs-backend/internal/config"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/gin-gonic/gin"
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
	api := r.engine.Group("/api/v1")
	{
		api.POST("/register", handlers.RegisterHandler(r.repository.User))
		api.POST("/login", handlers.LoginHandler(r.repository.User, r.config))
	}
}
