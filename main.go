package main

import (
	"i-lov-pdf/routes"

	"github.com/gin-gonic/gin"
)

func main() {

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.Static("/outputs", "./outputs")

	routes.SetUpRoutes(r)

	r.Run(":8000")

}
