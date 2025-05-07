package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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

type ServerState struct {
	mu               sync.Mutex
	Clients          map[string]*ClientState
	ExpectedUsers    int
	CurrentImagePath string
	OverallState     string // "WaitingForUsers", "WaitingForReady", "Triggered", "Displaying"
	TargetShowTime   string // RFC3339Nano format
}

func countReadyClients(state *ServerState) int {
	count := 0
	for _, client := range state.Clients {
		if client.IsReady {
			count++
		}
	}
	return count
}

func broadcastToAll(state *ServerState, message interface{}) {
	state.mu.Lock()
	defer state.mu.Unlock()

	for _, client := range state.Clients {
		if client.Conn != nil {
			client.Conn.WriteJSON(message)
		}
	}
}

type ClientState struct {
	Conn     *websocket.Conn
	IsReady  bool
	LastSeen int64
}

func setupServerState() *ServerState {
	return &ServerState{
		Clients:       make(map[string]*ClientState),
		ExpectedUsers: 10,
		OverallState:  "WaitingForUsers",
	}
}

func setupRouter(state *ServerState) *gin.Engine {
	r := gin.Default()

	r.POST("/upload", createUploadHandler(state))
	r.GET("/image", createImageHandler(state))
	r.GET("/ws", createWebSocketHandler(state))

	return r
}

func createUploadHandler(state *ServerState) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create uploads directory if it doesn't exist
		if err := os.MkdirAll("uploads", 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
			return
		}

		// Get the uploaded file
		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
			return
		}

		// Save the file to uploads/current_image.png
		filePath := filepath.Join("uploads", "current_image.png")
		if err := c.SaveUploadedFile(file, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
			return
		}

		// Update server state
		state.mu.Lock()
		state.CurrentImagePath = filePath
		state.mu.Unlock()

		// Broadcast image update to all connected clients
		broadcastToAll(state, gin.H{
			"type": "image_updated",
		})

		c.JSON(http.StatusOK, gin.H{"message": "Image uploaded successfully"})
	}
}

func createImageHandler(state *ServerState) gin.HandlerFunc {
	return func(c *gin.Context) {
		state.mu.Lock()
		imagePath := state.CurrentImagePath
		state.mu.Unlock()

		if imagePath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "No image available"})
			return
		}

		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Image file not found"})
			return
		}

		c.File(imagePath)
	}
}

func handleClientRegistration(state *ServerState, clientID string, conn *websocket.Conn) {
	if _, exists := state.Clients[clientID]; !exists {
		state.Clients[clientID] = &ClientState{
			Conn:    conn,
			IsReady: false,
		}
		// Transition to WaitingForReady if we have enough users
		if len(state.Clients) >= state.ExpectedUsers && state.OverallState == "WaitingForUsers" {
			state.OverallState = "WaitingForReady"
		}
	} else {
		state.Clients[clientID].Conn = conn
		state.Clients[clientID].LastSeen = time.Now().Unix()
	}

	// Broadcast updated user count to all
	broadcastToAll(state, gin.H{
		"type":       "user_count",
		"totalCount": len(state.Clients),
	})
}

func handleReadyState(state *ServerState, clientID string, isReady bool) {
	state.mu.Lock()
	defer state.mu.Unlock()

	if client, exists := state.Clients[clientID]; exists {
		client.IsReady = isReady
		client.LastSeen = time.Now().Unix()
		readyCount := countReadyClients(state)

		// Check if all expected users are ready
		if readyCount >= state.ExpectedUsers && state.OverallState == "WaitingForReady" {
			state.OverallState = "Triggered"
			state.TargetShowTime = time.Now().UTC().Add(5 * time.Second).Format(time.RFC3339Nano)
			return // Let the broadcast happen outside the lock
		}
	}
}

func sendFullState(state *ServerState, conn *websocket.Conn) {
	state.mu.Lock()
	defer state.mu.Unlock()

	conn.WriteJSON(gin.H{
		"type": "full_state",
		"state": gin.H{
			"readyCount":    countReadyClients(state),
			"totalCount":    len(state.Clients),
			"overallState":  state.OverallState,
			"hasImage":      state.CurrentImagePath != "",
			"targetTimeUTC": state.TargetShowTime,
		},
	})
}

func createWebSocketHandler(state *ServerState) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to upgrade to WebSocket: %v", err)
			return
		}
		defer conn.Close()

		for {
			var msg struct {
				Type     string `json:"type"`
				ClientID string `json:"clientId"`
				IsReady  bool   `json:"isReady"`
			}

			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}

			switch msg.Type {
			case "register":
				handleClientRegistration(state, msg.ClientID, conn)
				sendFullState(state, conn)

			case "ready":
				handleReadyState(state, msg.ClientID, msg.IsReady)
				readyCount := countReadyClients(state)

				// Broadcast ready status
				broadcastToAll(state, gin.H{
					"type":       "partial_state",
					"clientId":   msg.ClientID,
					"isReady":    msg.IsReady,
					"readyCount": readyCount,
					"totalCount": len(state.Clients),
				})

				// Broadcast start if triggered
				if state.OverallState == "Triggered" {
					broadcastToAll(state, gin.H{
						"type":               "start",
						"targetTimestampUTC": state.TargetShowTime,
					})
				}
			}
		}
	}
}

func startServer(r *gin.Engine) {
	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func main() {
	state := setupServerState()
	r := setupRouter(state)
	startServer(r)
}
