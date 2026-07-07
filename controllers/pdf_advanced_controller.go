package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  Delete Pages
// ──────────────────────────────────────────────────────────

func DeletePagesHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	pages := strings.TrimSpace(c.DefaultPostForm("pages", ""))
	if pages == "" {
		sendError(c, 400, "Specify pages to delete, e.g. '1,3,5' or '2-4'")
		return
	}
	pageList := splitPages(pages)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="edited.pdf"`)
	c.Status(200)
	if err := services.DeletePages(rs, c.Writer, pageList); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "delete-pages", origName)
}

// ──────────────────────────────────────────────────────────
//  Reorder Pages
// ──────────────────────────────────────────────────────────

func ReorderPagesHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	order := strings.TrimSpace(c.DefaultPostForm("order", ""))
	if order == "" {
		sendError(c, 400, "Specify page order, e.g. '3,1,2,5,4'")
		return
	}
	pageOrder := splitPages(order)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="reordered.pdf"`)
	c.Status(200)
	if err := services.ReorderPages(rs, c.Writer, pageOrder); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "reorder-pages", origName)
}

// ──────────────────────────────────────────────────────────
//  Crop Pages
// ──────────────────────────────────────────────────────────

func CropPagesHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	box := strings.TrimSpace(c.DefaultPostForm("box", ""))
	if box == "" {
		sendError(c, 400, "Specify crop box, e.g. '[0 0 400 600]'")
		return
	}
	pages := splitPages(c.DefaultPostForm("pages", ""))

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="cropped.pdf"`)
	c.Status(200)
	if err := services.CropPages(rs, c.Writer, pages, box); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "crop", origName)
}

// ──────────────────────────────────────────────────────────
//  Insert Blank Pages
// ──────────────────────────────────────────────────────────

func InsertBlankHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	afterPages := splitPages(c.DefaultPostForm("after", ""))
	beforeFlag := c.DefaultPostForm("mode", "after") == "before"

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="with_blanks.pdf"`)
	c.Status(200)
	if err := services.InsertBlankPages(rs, c.Writer, afterPages, beforeFlag); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "insert-blank", origName)
}

// ──────────────────────────────────────────────────────────
//  Add Page Numbers
// ──────────────────────────────────────────────────────────

func AddPageNumbersHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	position := c.DefaultPostForm("position", "bottom-center")
	startNum := 1

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="numbered.pdf"`)
	c.Status(200)
	if err := services.AddPageNumbers(rs, c.Writer, position, startNum); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "page-numbers", origName)
}

// ──────────────────────────────────────────────────────────
//  Extract Text
// ──────────────────────────────────────────────────────────

func ExtractTextHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	pages, err := services.ExtractText(rs)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": origName,
		"pages":    pages,
		"count":    len(pages),
	})
	logOp(c, "extract-text", origName)
}

// ──────────────────────────────────────────────────────────
//  Extract Images
// ──────────────────────────────────────────────────────────

func ExtractImagesHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	images, err := services.ExtractImages(rs)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}
	if len(images) == 0 {
		sendError(c, 422, "No embedded images found")
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="extracted_images.zip"`)
	c.Status(200)
	if err := services.ZipImageData(images, c.Writer); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "extract-images", origName)
}

// ──────────────────────────────────────────────────────────
//  Page Count (JSON response)
// ──────────────────────────────────────────────────────────

func PageCountHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	count, err := services.GetPageCount(rs)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"filename": origName, "pages": count})
}

// ──────────────────────────────────────────────────────────
//  Helpers
// ──────────────────────────────────────────────────────────

func splitPages(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// parseJSON is a helper to parse JSON form field
func parseJSON(raw string, v interface{}) error {
	return json.Unmarshal([]byte(raw), v)
}
