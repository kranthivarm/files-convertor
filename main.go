package main

import (
	"log"
	"os"

	"i-lov-pdf/db"
	"i-lov-pdf/middleware"
	"i-lov-pdf/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := db.Connect(); err != nil {
		log.Printf(" DB unavailable (%v) — running without history", err)
	} else {
		if err := db.Migrate(); err != nil {
			log.Printf("DB migrate error: %v", err)
		}
	}

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// ── Fix 2: Guard against DoS ──────────────────────────
	//
	// MaxMultipartMemory controls how much Go keeps in RAM
	// during multipart parsing. Files beyond this threshold
	// are transparently spooled to OS temp files (/tmp),
	// backed by a kernel-managed file descriptor.
	//
	// 32 MB = reasonable default. Files > 32 MB get a temp
	// file instead of a heap allocation, preventing OOM.
	r.MaxMultipartMemory = 32 << 20 // 32 MB

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	routes.SetUpRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("Starting server on :%s", port)
	r.Run(":" + port)
}
