package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	pongDeadline  = 500 * time.Millisecond
	pingFrequency = 500 * time.Millisecond
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

func pongHandler(conn *websocket.Conn) func(string) error {
	return func(message string) error {
		conn.SetReadDeadline(time.Now().Add(pingFrequency + pongDeadline))
		go ping(conn)

		return nil
	}
}

func ping(conn *websocket.Conn) {
	time.Sleep(pingFrequency)
	conn.WriteMessage(websocket.PingMessage, nil)
}

func removeClient(clientID string) {
	serverState.Lock()
	_, exists := serverState.Clients[clientID]
	if !exists {
		log.Printf("ERROR: Removing non-existing client: '%s'", clientID)
		serverState.Unlock()
		return
	}

	delete(serverState.Clients, clientID)

	readyCount, totalCount := readyDataLocked()

	if totalCount < serverState.ExpectedUsers &&
		serverState.OverallState == "WaitingForReady" {
		serverState.OverallState = "WaitingForUsers"
	}

	serverState.Unlock()
	log.Printf("Removing ClientID: %s", clientID)

	broadcastPartialState(clientID, false, readyCount, totalCount)
}

func handleDisconnection(clientID string) {

	serverState.Lock()
	client, exists := serverState.Clients[clientID]

	if !exists {
		log.Printf("ERROR: Disconnecting non-existing ClientID: %s", clientID)
		serverState.Unlock() // Unlock before returning
		return
	}

	// If there's an existing removal timer, stop it.
	if client.RemovalTimer != nil {
		log.Printf("ERROR: Doubbled disconnection Timer for ClientID: %s", clientID)
		client.RemovalTimer.Stop()
	}

	client.IsReady = false
	readyCount, totalCount := readyDataLocked()

	// Schedule client removal
	client.RemovalTimer = time.AfterFunc(5*time.Second, func() {
		removeClient(clientID)
	})
	//log.Printf("Scheduled removal of ClientID %s in 5 seconds.", clientID)

	serverState.Unlock()

	log.Printf("Disconnecting ClientID: %s", clientID)

	broadcastPartialState(clientID, false, readyCount, totalCount)
}

func handleClientRegistration(clientID string, isReady bool, conn *websocket.Conn) {
	serverState.Lock()

	if client, exists := serverState.Clients[clientID]; exists {
		log.Printf("Reconnecting clientID: %s", clientID)
		// Client is reconnecting
		client.Conn = conn
		client.registrationTime = time.Now().UnixNano()
		client.IsReady = isReady

		// Cancel pending removal if any
		if client.RemovalTimer != nil {
			client.RemovalTimer.Stop()
			client.RemovalTimer = nil // Clear the timer
		}
	} else {
		log.Printf("Registering clientID: %s", clientID)
		// New client registration
		serverState.Clients[clientID] = &ClientState{
			Conn:             conn,
			registrationTime: time.Now().UnixNano(),
			IsReady:          isReady,
			// RemovalTimer is nil by default
		}

		if len(serverState.Clients) >= serverState.ExpectedUsers &&
			serverState.OverallState == "WaitingForUsers" {
			serverState.OverallState = "WaitingForReady"
		}
	} // Get the actual state after registration logic

	readyCount, totalCount := readyDataLocked()

	serverState.Unlock() // Unlock before network I/O

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
	serverState.Lock()
	client, exists := serverState.Clients[clientID]
	if !exists {
		serverState.Unlock()
		log.Printf("Received 'ready' from unknown clientID: %s", clientID)
		return
	}

	client.IsReady = isReady

	readyCount, totalCount := readyDataLocked()

	parsedTargetTime := checkStartLocked(readyCount, totalCount)
	serverState.Unlock()

	if parsedTargetTime != "" {
		start(parsedTargetTime)
	} else {
		broadcastPartialState(clientID, isReady, readyCount, totalCount)
	}
}

func listenSocket(conn *websocket.Conn, clientID string) { // clientID will be passed after registration
	// Ping-pong handling
	conn.SetPongHandler(pongHandler(conn))
	go ping(conn)

	for {
		var msg struct {
			Type    string `json:"type"`
			IsReady bool   `json:"isReady"`
		}

		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for ClientID %s: %v, ", clientID, err)
			} else if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("Client %s disconnected normally: %v", clientID, err)
			} else {
				log.Printf("Read error for ClientID %s: %v.", clientID, err)
			}
			// TODO See error codes to treat acordingly
			break
		}

		switch msg.Type {
		case "ready":
			handleReadyState(clientID, msg.IsReady)
		default:
			log.Printf("Received unknown message type: %s from ClientID: %s", msg.Type, clientID)
		}
	}

	handleDisconnection(clientID)
}

func webSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// Expect the first message to be "register"
	var firstMsg struct {
		Type     string `json:"type"`
		ClientID string `json:"clientId"`
		IsReady  bool   `json:"isReady"`
	}

	if err := conn.ReadJSON(&firstMsg); err != nil {
		log.Printf("Error reading first message (expected register): %v", err)
		conn.Close()
		return
	}

	if firstMsg.Type != "register" {
		log.Printf("ERROR: First message was not 'register', received: %s. Closing connection.", firstMsg.Type)
		conn.WriteJSON(gin.H{"error": "First message must be of type 'register'"})
		conn.Close()
		return
	}

	handleClientRegistration(firstMsg.ClientID, firstMsg.IsReady, conn)

	go listenSocket(conn, firstMsg.ClientID)
}
