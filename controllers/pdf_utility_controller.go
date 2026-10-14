package controllers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  Compare two PDFs
// ──────────────────────────────────────────────────────────

func ComparePDFsHandler(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	fhs := form.File["files"]
	if len(fhs) < 2 {
		sendError(c, 400, "Upload exactly 2 PDF files to compare")
		return
	}

	h1, err := readOneFromHeader(fhs[0])
	if err != nil {
		sendError(c, 400, "Could not read first file: "+err.Error())
		return
	}
	defer h1.Close()

	h2, err := readOneFromHeader(fhs[1])
	if err != nil {
		sendError(c, 400, "Could not read second file: "+err.Error())
		return
	}
	defer h2.Close()

	result, err := services.ComparePDFs(h1, h2)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
	logOp(c, "compare", fhs[0].Filename)
}

// ──────────────────────────────────────────────────────────
//  Repair PDF
// ──────────────────────────────────────────────────────────

func RepairPDFHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	safeStream(c, "application/pdf", "repaired.pdf", func(w io.Writer) error {
		return services.RepairPDF(handle, w)
	})
	logOp(c, "repair", handle.Name)
}
