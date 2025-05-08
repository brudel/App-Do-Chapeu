package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
)

// countReadyClientsLocked assumes the globalState.mu is already locked.
func countReadyClientsLocked() int {
	count := 0
	for _, client := range globalState.Clients {
		if client.IsReady {
			count++
		}
	}
	return count
}

func broadcastToClients(message interface{}) {
	globalState.mu.Lock()
	// Create a list of connections to send to, under lock, to avoid holding lock during I/O
	connsToBroadcast := make([]*websocket.Conn, 0, len(globalState.Clients))
	for _, client := range globalState.Clients {
		if client.Conn != nil {
			connsToBroadcast = append(connsToBroadcast, client.Conn)
		}
	}
	globalState.mu.Unlock() // Unlock before sending

	for _, conn := range connsToBroadcast {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Error broadcasting to client: %v", err)
			// Consider more sophisticated error handling, e.g., removing dead clients.
		}
	}
}

func broadcastPartialState(clientID string, isReady bool, readyCount int, totalCount int) {
	broadcastToClients(gin.H{
		"type":       "partial_state",
		"clientId":   clientID,
		"isReady":    isReady,
		"readyCount": readyCount,
		"totalCount": totalCount,
	})
}

func handleClientRegistration(clientID string, conn *websocket.Conn) {
	globalState.mu.Lock()

	if _, exists := globalState.Clients[clientID]; !exists {
		globalState.Clients[clientID] = &ClientState{
			Conn:     conn,
			LastSeen: time.Now().Unix(),
		}
		if len(globalState.Clients) >= globalState.ExpectedUsers &&
			globalState.OverallState == "WaitingForUsers" {
			globalState.OverallState = "WaitingForReady"
		}
	} else {
		globalState.Clients[clientID].Conn = conn
		globalState.Clients[clientID].LastSeen = time.Now().Unix()
		globalState.Clients[clientID].IsReady = false // Reset ready state on re-register
	}
	clientIsReady := globalState.Clients[clientID].IsReady // Get the actual state after registration logic

	// Get counts for broadcastPartialState while lock is held
	currentReadyCount := countReadyClientsLocked()
	currentTotalCount := len(globalState.Clients)

	globalState.mu.Unlock() // Unlock before network I/O

	sendFullState(conn) // sendFullState handles its own locking
	broadcastPartialState(clientID, clientIsReady, currentReadyCount, currentTotalCount)
}

func handleReadyState(clientID string, isReady bool) {
	var startMessage gin.H

	globalState.mu.Lock()
	client, exists := globalState.Clients[clientID]
	if !exists {
		globalState.mu.Unlock()
		return
	}

	client.IsReady = isReady
	client.LastSeen = time.Now().Unix()

	currentReadyCount := countReadyClientsLocked()
	currentTotalCount := len(globalState.Clients)

	if currentReadyCount >= currentTotalCount &&
		globalState.OverallState == "WaitingForReady" {
		globalState.OverallState = "Triggered"
		globalState.TargetShowTime = time.Now().UTC().Add(3 * time.Second).Format(time.RFC3339Nano)
		startMessage = gin.H{
			"type":               "start",
			"targetTimestampUTC": globalState.TargetShowTime,
		}
	}
	globalState.mu.Unlock()

	broadcastPartialState(clientID, isReady, currentReadyCount, currentTotalCount)

	if startMessage != nil {
		broadcastToClients(startMessage)
	}
}

func sendFullState(conn *websocket.Conn) {
	globalState.mu.Lock()
	// Make it an struct
	readyCount := countReadyClientsLocked()
	totalCount := len(globalState.Clients)
	overallState := globalState.OverallState
	targetTime := globalState.TargetShowTime

	globalState.mu.Unlock() // Unlock before file access and network I/O

	hasImg := fileExists(imagePath) // fileExists is from image_handlers.go

	// Error from WriteJSON is not handled here, consider adding it.
	conn.WriteJSON(gin.H{
		"type": "full_state",
		"state": gin.H{
			"readyCount":    readyCount,
			"totalCount":    totalCount,
			"overallState":  overallState,
			"hasImage":      hasImg,
			"targetTimeUTC": targetTime,
		},
	})
}

func webSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// On initial connection, we don't know the clientID yet.
	// It will be sent in the "register" message.

	for {
		var msg struct {
			Type     string `json:"type"`
			ClientID string `json:"clientId"`
			IsReady  bool   `json:"isReady"` // Used by "ready" type
		}

		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v, ClientID: %s", err, msg.ClientID)
			}
			// TODO: Handle client disconnection: remove client from globalState.Clients
			// This needs locking and careful handling if msg.ClientID is known.
			break
		}

		globalState.mu.Lock() // Lock for operations that might read/write client map or client state
		client, clientExists := globalState.Clients[msg.ClientID]
		if clientExists {
			client.LastSeen = time.Now().Unix()
		}
		globalState.mu.Unlock() // Unlock after updating LastSeen

		switch msg.Type {
		case "register":
			handleClientRegistration(msg.ClientID, conn)
		case "ready":
			if !clientExists { // Ensure client is registered before processing ready state
				log.Printf("Received 'ready' from unknown clientID: %s", msg.ClientID)
				continue
			}
			handleReadyState(msg.ClientID, msg.IsReady)
		default:
			log.Printf("Received unknown message type: %s from ClientID: %s", msg.Type, msg.ClientID)
		}
	}
	// TODO: Implement client cleanup logic here if ClientID was established.
	// For example, after the loop breaks, if a clientID was associated with this connection,
	// remove it from globalState.Clients and broadcast an update.
	// This requires knowing the clientID for this connection.
}
