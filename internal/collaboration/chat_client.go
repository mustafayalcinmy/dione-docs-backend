package collaboration

import (
	"log"
	"time"

	"github.com/dione-docs-backend/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type ChatClient struct {
	userID uuid.UUID
	hub    *ChatHub
	conn   *websocket.Conn
	send   chan *models.Message
}

func NewChatClient(hub *ChatHub, conn *websocket.Conn, userID uuid.UUID) {
	client := &ChatClient{
		userID: userID,
		hub:    hub,
		conn:   conn,
		send:   make(chan *models.Message, 256),
	}
	client.hub.Register <- client

	go client.writePump()
	go client.readPump()
}

func (c *ChatClient) readPump() {
	defer func() {
		c.hub.Unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		var msg IncomingMessage
		err := c.conn.ReadJSON(&msg)

		if err != nil {
			log.Printf("FATAL: readPump Error for client %s. The connection will be closed. Error: %v", c.userID, err)

			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error type was an UnexpectedCloseError.")
			}
			break
		}

		log.Printf("Message received from client %s: %s", c.userID, msg.Content)
		c.hub.processAndBroadcast(msg, c.userID)
	}
}
func (c *ChatClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteJSON(message)
			if err != nil {
				log.Printf("Error writing json to chat client: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
