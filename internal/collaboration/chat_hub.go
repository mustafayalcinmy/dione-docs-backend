package collaboration

import (
	"log"
	"sync"
	"time"

	"github.com/dione-docs-backend/internal/models"
	"github.com/dione-docs-backend/internal/repository"
	"github.com/google/uuid"
)

type IncomingMessage struct {
	Content string `json:"content"`
}

type ChatHub struct {
	docID      uuid.UUID
	clients    map[*ChatClient]bool
	Register   chan *ChatClient
	Unregister chan *ChatClient
	broadcast  chan *models.Message
	mu         sync.Mutex
	repo       *repository.Repository
}

func NewChatHub(docID uuid.UUID, repo *repository.Repository) *ChatHub {
	return &ChatHub{
		docID:      docID,
		clients:    make(map[*ChatClient]bool),
		Register:   make(chan *ChatClient),
		Unregister: make(chan *ChatClient),
		broadcast:  make(chan *models.Message, 5),
		repo:       repo,
	}
}

func (h *ChatHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			log.Printf("Chat Client %s connected to hub for doc %s", client.userID, h.docID)
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("Chat Client %s disconnected from hub for doc %s", client.userID, h.docID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *ChatHub) processAndBroadcast(incomingMsg IncomingMessage, userID uuid.UUID) {
	log.Println("--- [DEBUG] processAndBroadcast: Function started. ---")
	log.Printf("[DEBUG] Incoming message: '%s' from user: %s for doc: %s", incomingMsg.Content, userID, h.docID)

	log.Printf("[DEBUG] Step 1: Finding document by ID: %s", h.docID)
	var doc models.Document
	if err := h.repo.Document.GetByID(h.docID, &doc); err != nil {
		log.Printf("[DEBUG] FATAL: Could not find document. Error: %v", err)
		log.Println("--- [DEBUG] processAndBroadcast: Exiting due to document not found. ---")
		return
	}

	isOwner := doc.OwnerID == userID
	canWrite := isOwner

	if !isOwner {
		log.Println("[DEBUG] Step 3: User is not owner. Checking permissions...")
		permission, err := h.repo.Permission.GetAcceptedByDocumentAndUser(h.docID, userID)
		if err != nil {
			log.Printf("[DEBUG] FATAL: Permission check failed for non-owner. Error: %v", err)
			log.Println("--- [DEBUG] processAndBroadcast: Exiting due to permission check error. ---")
			return
		}

		if permission.AccessType == "editor" {
			canWrite = true
			log.Println("[DEBUG] User has 'editor' access. Setting canWrite to true.")
		}
	}

	log.Printf("[DEBUG] Step 4: Final check. Can user write? %t", canWrite)
	if !canWrite {
		log.Println("[DEBUG] FATAL: User does not have write permission.")
		log.Println("--- [DEBUG] processAndBroadcast: Exiting because canWrite is false. ---")
		return
	}

	log.Println("[DEBUG] PERMISSION OK. Proceeding to process message.")

	log.Printf("[DEBUG] Step 5: Finding user details by ID: %s", userID)
	var user models.User
	if err := h.repo.User.GetByID(userID, &user); err != nil {
		log.Printf("[DEBUG] FATAL: Could not find user details in database. Error: %v", err)
		log.Println("--- [DEBUG] processAndBroadcast: Exiting because user could not be found. ---")
		return
	}

	dbMessage := &models.Message{
		ID:         uuid.New(),
		DocumentID: h.docID,
		UserID:     userID,
		UserName:   user.Username,
		Content:    incomingMsg.Content,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.repo.Message.Create(dbMessage); err != nil {
		log.Printf("[DEBUG] FATAL: Error saving message to DB. Error: %v", err)
		log.Println("--- [DEBUG] processAndBroadcast: Exiting because DB save failed. ---")
		return
	}

	dbMessage.User = user

	h.broadcast <- dbMessage
}
