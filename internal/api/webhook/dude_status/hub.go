package dude_status

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
)

const clientSendBufferSize = 16

type wsClient struct {
	connection *websocket.Conn
	send       chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
}

var (
	sharedHubMu sync.RWMutex
	sharedHub   *Hub
)

func GetSharedHub() *Hub {
	sharedHubMu.RLock()
	defer sharedHubMu.RUnlock()
	return sharedHub
}

func SetSharedHub(h *Hub) {
	sharedHubMu.Lock()
	defer sharedHubMu.Unlock()
	sharedHub = h
}

func NewHub() *Hub {
	h := &Hub{clients: make(map[*wsClient]struct{})}
	SetSharedHub(h)
	return h
}

func (h *Hub) register(client *wsClient) {
	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(client *wsClient) {
	h.mu.Lock()
	if _, connected := h.clients[client]; connected {
		delete(h.clients, client)
		close(client.send)
	}
	h.mu.Unlock()
}

func (h *Hub) Broadcast(status DeviceStatus) error {
	message, err := json.Marshal(status)
	if err != nil {
		return err
	}

	var slowClients []*wsClient
	h.mu.RLock()
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			slowClients = append(slowClients, client)
		}
	}
	h.mu.RUnlock()

	for _, client := range slowClients {
		h.unregister(client)
		log.Printf("[dude-websocket] disconnected slow client")
	}
	return nil
}

func (h *Hub) HandleConnection(connection *websocket.Conn) {
	client := &wsClient{
		connection: connection,
		send:       make(chan []byte, clientSendBufferSize),
	}
	h.register(client)
	defer func() {
		h.unregister(client)
		_ = connection.Close()
	}()

	go h.writeMessages(client)
	for {
		if _, _, err := connection.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writeMessages(client *wsClient) {
	defer func() {
		_ = client.connection.Close()
	}()
	for message := range client.send {
		if err := client.connection.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}
