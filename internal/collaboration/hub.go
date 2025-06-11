// internal/collaboration/hub.go
package collaboration

import (
	"log"
	"sync"

	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/google/uuid"
)

type OTOperation struct {
	Version  int           `json:"version"`
	ClientID string        `json:"clientId"`
	Ops      []interface{} `json:"ops"`
}

type Hub struct {
	docID         uuid.UUID
	clients       map[*Client]bool
	Register      chan *Client
	Unregister    chan *Client
	broadcast     chan OTOperation
	mu            sync.Mutex
	docRepo       repository.DocumentRepository
	documentState []byte
	version       int
	history       []OTOperation
}

func NewHub(docID uuid.UUID, docRepo repository.DocumentRepository) *Hub {
	var doc models.Document
	if err := docRepo.GetByID(docID, &doc); err != nil {
		log.Printf("Error getting document for hub %s: %v. Starting with empty doc.", docID, err)
		return &Hub{
			docID:         docID,
			clients:       make(map[*Client]bool),
			Register:      make(chan *Client),
			Unregister:    make(chan *Client),
			broadcast:     make(chan OTOperation, 5),
			docRepo:       docRepo,
			documentState: []byte(`{"ops":[{"insert":"\n"}]}`),
			version:       1,
			history:       make([]OTOperation, 0),
		}
	}

	return &Hub{
		docID:         docID,
		clients:       make(map[*Client]bool),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		broadcast:     make(chan OTOperation, 5),
		docRepo:       docRepo,
		documentState: doc.Content,
		version:       doc.Version,
		history:       make([]OTOperation, 0),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			log.Printf("Client %s connected to hub for doc %s", client.ID, h.docID)
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Client %s disconnected from hub for doc %s", client.ID, h.docID)
			}
			h.mu.Unlock()

		case operation := <-h.broadcast:
			h.mu.Lock()
			h.version++
			operation.Version = h.version
			h.history = append(h.history, operation)
			for client := range h.clients {
				if client.ID != operation.ClientID {
					select {
					case client.send <- operation:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}
