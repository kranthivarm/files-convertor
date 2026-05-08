package main

import (
	"i-lov-pdf/routes"

	"github.com/gin-gonic/gin"
)

func main(){

	r := gin.Default()
	
	routes.SetUpRoutes(r)
	
	r.Run(":8000");
	
}