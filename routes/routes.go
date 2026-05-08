package routes

import (
	"i-lov-pdf/controllers"

	"github.com/gin-gonic/gin"
)

func SetUpRoutes(r *gin.Engine) {

	r.GET("/", controllers.Home)

	api := r.Group("/api")
	{
		api.POST("/merge", controllers.Merge)
		api.POST("/split", controllers.Split)
		api.POST("/compress", controllers.Compress)
		api.POST("/rotate", controllers.Rotate)
		api.POST("/watermark", controllers.Watermark)
		api.POST("/jpg-to-pdf", controllers.JPGToPDF)
		api.POST("/pdf-to-jpg", controllers.PDFToJPG)
	}
}
