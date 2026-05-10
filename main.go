package main

import (
	"i-lov-pdf/cleanup"
	"i-lov-pdf/routes"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	os.MkdirAll("uploads", 0755)
	os.MkdirAll("outputs", 0755)

	cleanup.Start("uploads", "outputs")
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.Static("/outputs", "./outputs")

	routes.SetUpRoutes(r)

	r.Run(":8000")

}
