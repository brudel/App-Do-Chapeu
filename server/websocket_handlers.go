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

func handleClientRegistration(clientID string, isReady bool, conn *websocket.Conn) {
	serverState.mu.Lock()

	if _, exists := serverState.Clients[clientID]; exists {
		serverState.Clients[clientID].Conn = conn
		serverState.Clients[clientID].LastSeen = time.Now().Unix()
		serverState.Clients[clientID].IsReady = isReady
	} else {
		serverState.Clients[clientID] = &ClientState{
			Conn:     conn,
			LastSeen: time.Now().Unix(),
			IsReady:  isReady,
		}

		if len(serverState.Clients) >= serverState.ExpectedUsers &&
			serverState.OverallState == "WaitingForUsers" {
			serverState.OverallState = "WaitingForReady"
		}
	} // Get the actual state after registration logic

	// Get counts for broadcastPartialState while lock is held
	readyCount := countReadyClientsLocked()
	totalCount := len(serverState.Clients)

	serverState.mu.Unlock() // Unlock before network I/O

	conn.WriteJSON(generateFullStateMessage()) // sendFullState handles its own locking
	broadcastPartialState(clientID, isReady, readyCount, totalCount)
}

func checkStartLocked(readyCount int, totalCount int) string {
	if readyCount < totalCount ||
		serverState.OverallState != "WaitingForReady" {
		return ""
	}

	serverState.OverallState = "Triggered"
	serverState.TargetShowTime = time.Now().UTC().Add(3 * time.Second).Format(time.RFC3339Nano)

	return serverState.TargetShowTime
}

func handleReadyState(clientID string, isReady bool) {
	serverState.mu.Lock()
	client, exists := serverState.Clients[clientID]
	if !exists {
		serverState.mu.Unlock()
		log.Printf("Received 'ready' from unknown clientID: %s", clientID)
		return
	}

	client.IsReady = isReady
	client.LastSeen = time.Now().Unix()

	readyCount := countReadyClientsLocked()
	totalCount := len(serverState.Clients)

	parsedTargetTime := checkStartLocked(readyCount, totalCount)
	serverState.mu.Unlock()

	if parsedTargetTime != "" {
		start(readyCount, totalCount, parsedTargetTime)
	} else {
		broadcastPartialState(clientID, isReady, readyCount, totalCount)
	}
}

func listenSocket(conn *websocket.Conn) {
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
		switch msg.Type {
		case "register":
			handleClientRegistration(msg.ClientID, msg.IsReady, conn)
		case "ready":
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

func webSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// On initial connection, we don't know the clientID yet.
	// It will be sent in the "register" message.

	go listenSocket(conn)
}
