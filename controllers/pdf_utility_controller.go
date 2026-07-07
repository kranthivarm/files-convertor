package controllers

import (
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

	rs1, _, err := readOneFromHeader(fhs[0])
	if err != nil {
		sendError(c, 400, "Could not read first file: "+err.Error())
		return
	}
	rs2, _, err := readOneFromHeader(fhs[1])
	if err != nil {
		sendError(c, 400, "Could not read second file: "+err.Error())
		return
	}

	result, err := services.ComparePDFs(rs1, rs2)
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
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="repaired.pdf"`)
	c.Status(200)
	if err := services.RepairPDF(rs, c.Writer); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "repair", origName)
}
