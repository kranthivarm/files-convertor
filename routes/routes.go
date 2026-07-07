package routes

import (
	"i-lov-pdf/controllers"
	"i-lov-pdf/middleware"

	"github.com/gin-gonic/gin"
)

func SetUpRoutes(r *gin.Engine) {

	r.GET("/", controllers.Home)

	api := r.Group("/api")
	api.Use(middleware.OptionalAuth())
	{
		// ── Basic PDF Operations ──────────────────────────────
		api.POST("/merge", controllers.Merge)
		api.POST("/split", controllers.Split)
		api.POST("/compress", controllers.Compress)
		api.POST("/rotate", controllers.Rotate)
		api.POST("/watermark", controllers.Watermark)
		api.POST("/jpg-to-pdf", controllers.JPGToPDF)
		api.POST("/pdf-to-jpg", controllers.PDFToJPG)

		// ── Page Editing ─────────────────────────────────────
		api.POST("/delete-pages", controllers.DeletePagesHandler)
		api.POST("/reorder-pages", controllers.ReorderPagesHandler)
		api.POST("/crop", controllers.CropPagesHandler)
		api.POST("/insert-blank", controllers.InsertBlankHandler)
		api.POST("/page-numbers", controllers.AddPageNumbersHandler)
		api.POST("/extract-text", controllers.ExtractTextHandler)
		api.POST("/extract-images", controllers.ExtractImagesHandler)
		api.POST("/page-count", controllers.PageCountHandler)

		// ── Security ─────────────────────────────────────────
		api.POST("/encrypt", controllers.EncryptPDFHandler)
		api.POST("/decrypt", controllers.DecryptPDFHandler)
		api.POST("/redact", controllers.RedactPDFHandler)
		api.POST("/sign-pdf", controllers.SignPDFHandler)

		// ── Forms ────────────────────────────────────────────
		api.POST("/form-fields", controllers.ListFormFieldsHandler)
		api.POST("/fill-form", controllers.FillFormHandler)
		api.POST("/flatten-form", controllers.FlattenFormHandler)
		api.POST("/export-form", controllers.ExportFormHandler)

		// ── Utility ──────────────────────────────────────────
		api.POST("/compare", controllers.ComparePDFsHandler)
		api.POST("/repair", controllers.RepairPDFHandler)
		api.POST("/protect-files", controllers.ProtectFilesHandler)

		// ── Auth — Email+Password ────────────────────────────
		api.POST("/register", controllers.Register)
		api.POST("/login", controllers.Login)
		api.POST("/logout", controllers.Logout)

		// ── Auth — OTP ───────────────────────────────────────
		api.POST("/auth/otp/send", controllers.OTPSend)
		api.POST("/auth/otp/verify", controllers.OTPVerify)

		// ── Auth — OAuth ─────────────────────────────────────
		api.POST("/auth/google", controllers.GoogleAuth)
		api.POST("/auth/apple", controllers.AppleAuth)
	}

	// Protected routes (require authentication)
	auth := r.Group("/api")
	auth.Use(middleware.RequireAuth())
	{
		auth.GET("/me", controllers.Me)
		auth.GET("/history", controllers.History)
	}
}
