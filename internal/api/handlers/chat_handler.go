// file: internal/api/handlers/chat_handler.go

package handlers

import (
	"log"
	"net/http"
	"sync"

	"github.com/dione-docs-backend/internal/collaboration"
	"github.com/dione-docs-backend/internal/config"
	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/dione-docs-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ChatHandler struct {
	repo       *repository.Repository
	hubManager *ChatHubManager
	wsUpgrader websocket.Upgrader
	config     *config.Config
}

type ChatHubManager struct {
	hubs map[uuid.UUID]*collaboration.ChatHub
	mu   sync.Mutex
	repo *repository.Repository
}

func NewChatHubManager(repo *repository.Repository) *ChatHubManager {
	return &ChatHubManager{
		hubs: make(map[uuid.UUID]*collaboration.ChatHub),
		repo: repo,
	}
}

func (m *ChatHubManager) GetOrCreateHub(docID uuid.UUID) *collaboration.ChatHub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[docID]; ok {
		return hub
	}

	hub := collaboration.NewChatHub(docID, m.repo)
	m.hubs[docID] = hub
	go hub.Run()
	return hub
}

func NewChatHandler(repo *repository.Repository, hubManager *ChatHubManager, cfg *config.Config) *ChatHandler {
	return &ChatHandler{
		repo:       repo,
		hubManager: hubManager,
		config:     cfg,
		wsUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *ChatHandler) ServeChatWs(c *gin.Context) {
	log.Println("--- [CHAT-HANDLER-DEBUG] ServeChatWs: Connection request received. ---")

	tokenString := c.Query("token")
	if tokenString == "" {
		log.Println("[CHAT-HANDLER-DEBUG] FATAL: Token not found in query. Responding with 401.")
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authorization token not provided"})
		return
	}

	claims := &utils.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.config.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		log.Printf("[CHAT-HANDLER-DEBUG] FATAL: Token is invalid. Error: %v. Responding with 401.", err)
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid or expired token"})
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		log.Printf("[CHAT-HANDLER-DEBUG] FATAL: UserID in token is not a valid UUID. Error: %v. Responding with 400.", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid user identifier in token"})
		return
	}

	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		log.Printf("[CHAT-HANDLER-DEBUG] FATAL: DocumentID in path is not a valid UUID. Error: %v. Responding with 400.", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid document ID"})
		return
	}

	log.Println("[CHAT-HANDLER-DEBUG] Step 5: Checking document access permissions...")
	if !h.canUserAccessDocument(userID, docID) {
		log.Printf("[CHAT-HANDLER-DEBUG] FATAL: canUserAccessDocument returned false. User %s denied access to doc %s. Responding with 403.", userID, docID)
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Access to this document is denied"})
		return
	}

	hub := h.hubManager.GetOrCreateHub(docID)

	conn, err := h.wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[CHAT-HANDLER-DEBUG] FATAL: Failed to upgrade connection to WebSocket. Error: %v", err)
		return
	}

	collaboration.NewChatClient(hub, conn, userID)
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid document ID"})
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Authentication required"})
		return
	}

	if !h.canUserAccessDocument(userID, docID) {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "Access to this document is denied"})
		return
	}

	messages, err := h.repo.Message.GetByDocumentID(docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func (h *ChatHandler) canUserAccessDocument(userID, docID uuid.UUID) bool {
	var doc models.Document
	if err := h.repo.Document.GetByID(docID, &doc); err != nil {
		return false
	}

	if doc.OwnerID == userID || doc.IsPublic {
		return true
	}

	permission, err := h.repo.Permission.GetAcceptedByDocumentAndUser(docID, userID)
	if err != nil || permission == nil {
		return false
	}

	return true
}
