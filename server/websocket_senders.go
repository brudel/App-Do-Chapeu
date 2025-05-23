package main

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func readyDataLocked() (readyCount int, totalCount int) {
	readyCount = 0
	for _, client := range serverState.Clients {
		if client.IsReady {
			readyCount++
		}
	}

	return readyCount, len(serverState.Clients)
}

func broadcastToClients(message interface{}) {
	serverState.Lock()

	connsToBroadcast := make([]*websocket.Conn, 0, len(serverState.Clients))
	for _, client := range serverState.Clients {
		if client.Conn != nil {
			connsToBroadcast = append(connsToBroadcast, client.Conn)
		}
	}
	serverState.Unlock()

	for _, conn := range connsToBroadcast {
		go func() {
			if err := conn.WriteJSON(message); err != nil {
				log.Printf("Error broadcasting to client: %v", err)
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
	serverState.Lock()
	// Make it an struct
	readyCount, totalCount := readyDataLocked()
	overallState := serverState.OverallState
	targetTime := serverState.TargetShowTime

	serverState.Unlock() // Unlock before file access and network I/O

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

func start(targetTime string) {
	broadcastToClients(gin.H{
		"type":               "start",
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
