package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const imagePath = "uploads/current_image.png"

// fileExists checks if a file exists and is not a directory.
func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		// os.IsNotExist(err) is true if the file does not exist.
		// Other errors mean we can't determine existence, so treat as not existing or problematic.
		return false
	}
	return !info.IsDir() // File exists, return true if it's not a directory
}

func uploadHandler(c *gin.Context) {
	if err := os.MkdirAll("uploads", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	if err := c.SaveUploadedFile(file, imagePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

	broadcastToClients(gin.H{"type": "image_updated"}) // This will call the function in websocket_handlers.go
	c.JSON(http.StatusOK, gin.H{"message": "Image uploaded successfully"})
}

func imageHandler(c *gin.Context) {
	if !fileExists(imagePath) { // Use the refactored fileExists
		c.JSON(http.StatusNotFound, gin.H{"error": "Image file not found"})
		return
	}
	c.File(imagePath)
}
