package main

import (
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const minUsers = 1
const serverResetDelay = 3 * time.Second

type ServerState struct {
	Clients        map[string]*ClientState
	ExpectedUsers  int
	OverallState   string // "WaitingForUsers", "WaitingForReady", "Triggered", "Displaying"
	TargetShowTime string // RFC3339Nano format
	sync.Mutex
}

type ClientState struct {
	Conn             *websocket.Conn
	IsReady          bool
	registrationTime int64
	RemovalTimer     *time.Timer
	sync.Mutex
}

var serverState ServerState

func softStateReset() {

	serverState.Lock()

	serverState.OverallState = "WaitingForReady" // Reset to a known initial state
	serverState.TargetShowTime = ""              // Clear the target time as it's now processed

	for clientID := range serverState.Clients {
		if client, ok := serverState.Clients[clientID]; ok {
			client.IsReady = false // Reset ready state for all clients
		}
	}
	log.Printf("AfterFunc: Global state has been reset. New state: %s.", serverState.OverallState)

	serverState.Unlock() // Unlock BEFORE I/O (fileExists and broadcasting)

	broadcastToClients(generateFullStateMessage())
}

func main() {
	serverState = ServerState{
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
