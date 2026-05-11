package main

import (
	"log"
	"i-lov-pdf/cleanup"
	"i-lov-pdf/routes"
	"i-lov-pdf/db"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	os.MkdirAll("uploads", 0755)
	os.MkdirAll("outputs", 0755)

	if err := db.Connect(); err != nil {
		log.Printf("⚠️  DB unavailable (%v) — running without history", err)
	} else {
		if err := db.Migrate(); err != nil {
			log.Printf("⚠️  DB migrate error: %v", err)
		}
	}
	

	cleanup.Start("uploads", "outputs")
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.Static("/outputs", "./outputs")

	routes.SetUpRoutes(r)

	r.Run(":8000")

}
