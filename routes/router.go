package routes

import "github.com/gin-gonic/gin"

func SetUpRoutes(r *gin.Engine){
	r.LoadHTMLGlob("templates/*")

	// Static folders
	r.Static("/static", "./static")
	r.Static("/outputs", "./outputs")

	// Home route
	r.GET("/", handlers.Home)

	// API group
	api := r.Group("/api")
	{
		api.POST("/merge", handlers.Merge)
		api.POST("/split", handlers.Split)
		api.POST("/compress", handlers.Compress)
		api.POST("/rotate", handlers.Rotate)
		api.POST("/watermark", handlers.Watermark)
		api.POST("/jpg-to-pdf", handlers.JPGToPDF)
		api.POST("/pdf-to-jpg", handlers.PDFToJPG)
	}
}