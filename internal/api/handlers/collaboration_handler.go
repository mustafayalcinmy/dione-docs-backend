package handlers

import (
	"log"
	"net/http"
	"sync"

	"github.com/dione-docs-backend/internal/collaboration"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type HubManager struct {
	hubs    map[uuid.UUID]*collaboration.Hub
	mu      sync.Mutex
	docRepo repository.DocumentRepository
}

func NewHubManager(docRepo repository.DocumentRepository) *HubManager {
	return &HubManager{
		hubs:    make(map[uuid.UUID]*collaboration.Hub),
		docRepo: docRepo,
	}
}

func (m *HubManager) GetOrCreateHub(docID uuid.UUID) *collaboration.Hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if hub, ok := m.hubs[docID]; ok {
		return hub
	}

	hub := collaboration.NewHub(docID, m.docRepo)
	m.hubs[docID] = hub
	go hub.Run()
	return hub
}

// ServeWs, websocket isteklerini yönetir.
func (m *HubManager) ServeWs(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	hub := m.GetOrCreateHub(docID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}

	clientID := uuid.New().String()

	// DÜZELTME: Artık client'ı manuel olarak oluşturmuyoruz.
	// Bunun yerine collaboration paketindeki NewClient fonksiyonunu çağırıyoruz.
	// Bu fonksiyon, client'ı oluşturup goroutine'lerini kendi içinde başlatacak.
	collaboration.NewClient(hub, conn, clientID)
}
