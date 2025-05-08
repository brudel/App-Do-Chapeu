package main

import (
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	// gorilla/websocket is now used in websocket_handlers.go
)

const minUsers = 1
const serverResetDelay = 3 * time.Second

type ServerState struct {
	mu             sync.Mutex
	Clients        map[string]*ClientState
	ExpectedUsers  int
	OverallState   string // "WaitingForUsers", "WaitingForReady", "Triggered", "Displaying"
	TargetShowTime string // RFC3339Nano format
}

type ClientState struct {
	Conn     *websocket.Conn
	IsReady  bool
	LastSeen int64
}

var globalState ServerState

func softStateReset() {

	globalState.mu.Lock()

	globalState.OverallState = "WaitingForReady" // Reset to a known initial state
	globalState.TargetShowTime = ""              // Clear the target time as it's now processed

	for clientID := range globalState.Clients {
		if client, ok := globalState.Clients[clientID]; ok {
			client.IsReady = false // Reset ready state for all clients
		}
	}
	log.Printf("AfterFunc: Global state has been reset. New state: %s.", globalState.OverallState)

	globalState.mu.Unlock() // Unlock BEFORE I/O (fileExists and broadcasting)

	broadcastToClients(generateFullStateMessage())
}

func main() {
	globalState = ServerState{
		Clients:       make(map[string]*ClientState),
		ExpectedUsers: minUsers,
		OverallState:  "WaitingForUsers",
	}

	r := gin.Default()

	r.POST("/image", uploadHandler)
	r.GET("/image", imageHandler)
	r.GET("/ws", webSocketHandler)

	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
