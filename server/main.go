package main

import (
	"log"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	// gorilla/websocket is now used in websocket_handlers.go
)

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

var globalState *ServerState

func main() {
	globalState = &ServerState{
		Clients:       make(map[string]*ClientState),
		ExpectedUsers: 1,
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
