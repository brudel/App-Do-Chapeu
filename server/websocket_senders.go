package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// countReadyClientsLocked assumes the globalState.mu is already locked.
func countReadyClientsLocked() int {
	count := 0
	for _, client := range serverState.Clients {
		if client.IsReady {
			count++
		}
	}
	return count
}

func broadcastToClients(message interface{}) {
	serverState.mu.Lock()
	// Create a list of connections to send to, under lock, to avoid holding lock during I/O
	connsToBroadcast := make([]*websocket.Conn, 0, len(serverState.Clients))
	for _, client := range serverState.Clients {
		if client.Conn != nil {
			connsToBroadcast = append(connsToBroadcast, client.Conn)
		}
	}
	serverState.mu.Unlock() // Unlock before sending

	for _, conn := range connsToBroadcast {
		go func() {
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("Error broadcasting to client: %v", err)
				// Consider more sophisticated error handling, e.g., removing dead clients.
			}
		}()
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

func generateFullStateMessage() gin.H {
	serverState.mu.Lock()
	// Make it an struct
	readyCount := countReadyClientsLocked()
	totalCount := len(serverState.Clients)
	overallState := serverState.OverallState
	targetTime := serverState.TargetShowTime

	serverState.mu.Unlock() // Unlock before file access and network I/O

	hasImg := fileExists(imagePath) // fileExists is from image_handlers.go
	return gin.H{
		"type": "full_state",
		"state": gin.H{
			"readyCount":    readyCount,
			"totalCount":    totalCount,
			"overallState":  overallState,
			"hasImage":      hasImg,
			"targetTimeUTC": targetTime,
		},
	}
}

func start(readyCount int, totalCount int, targetTime string) {
	// Broadcast the start message with the currently active target time
	broadcastToClients(gin.H{
		"type":               "start",
		"readyCount":         readyCount,
		"totalCount":         totalCount,
		"targetTimestampUTC": targetTime,
	})

	//log.Printf("start-goroutine: Scheduling reset for target time: %s", targetTime)

	parsedTargetTime, err := time.Parse(time.RFC3339Nano, targetTime)
	if err != nil {
		log.Printf("start-goroutine: Error parsing TargetShowTime ('%s') for reset: %v", targetTime, err)
		return
	}

	// Schedule the reset logic in a new goroutine.
	// Pass activeTargetTime to ensure the goroutine works with the value from this specific 'start' call.
	nowUTC := time.Now().UTC()
	duration := parsedTargetTime.Sub(nowUTC) + serverResetDelay

	// time.AfterFunc executes the provided function in a new goroutine after the specified duration.
	go time.AfterFunc(duration, func() {
		softStateReset() // Execute the goroutine, passing the captured target time
	})
}
