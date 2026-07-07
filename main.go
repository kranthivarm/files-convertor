package main

import (
	"log"
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
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	routes.SetUpRoutes(r)

	r.Run(":8000")
}
