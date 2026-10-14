package controllers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/services"
)

// VideoToolPage serves the video watermark removal HTML page.
func VideoToolPage(c *gin.Context) {
	c.HTML(http.StatusOK, "watermark_remove.html", nil)
}

// RemoveVideoWatermark handles POST /api/video/remove-watermark.
// Accepts a video file, automatically detects the watermark via
// frame-difference analysis, and returns the cleaned video.
func RemoveVideoWatermark(c *gin.Context) {
	fh, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video file is required: " + err.Error()})
		return
	}

	ext := filepath.Ext(fh.Filename)
	if ext == "" {
		ext = ".mp4"
	}

	// Open the uploaded file from the multipart form
	src, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read uploaded file: " + err.Error()})
		return
	}
	defer src.Close()

	// Write to a temp input file
	inputFile, err := os.CreateTemp("", "wm-input-*"+ext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create temp file: " + err.Error()})
		return
	}
	inputPath := inputFile.Name()
	defer os.Remove(inputPath)

	if _, err := io.Copy(inputFile, src); err != nil {
		inputFile.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not write uploaded file: " + err.Error()})
		return
	}
	inputFile.Close()

	// Create temp output file
	outputFile, err := os.CreateTemp("", "wm-output-*"+ext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create temp output file: " + err.Error()})
		return
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	// Auto-detect and remove watermark
	if err := services.AutoRemoveWatermark(inputPath, outputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Serve the processed video
	outputName := "cleaned_" + fh.Filename
	c.FileAttachment(outputPath, outputName)
}
