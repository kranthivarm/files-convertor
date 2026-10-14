package controllers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  Delete Pages
// ──────────────────────────────────────────────────────────

func DeletePagesHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	pages := strings.TrimSpace(c.DefaultPostForm("pages", ""))
	if pages == "" {
		sendError(c, 400, "Specify pages to delete, e.g. '1,3,5' or '2-4'")
		return
	}
	pageList := splitPages(pages)

	safeStream(c, "application/pdf", "edited.pdf", func(w io.Writer) error {
		return services.DeletePages(handle, w, pageList)
	})
	logOp(c, "delete-pages", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Reorder Pages
// ──────────────────────────────────────────────────────────

func ReorderPagesHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	order := strings.TrimSpace(c.DefaultPostForm("order", ""))
	if order == "" {
		sendError(c, 400, "Specify page order, e.g. '3,1,2,5,4'")
		return
	}
	pageOrder := splitPages(order)

	safeStream(c, "application/pdf", "reordered.pdf", func(w io.Writer) error {
		return services.ReorderPages(handle, w, pageOrder)
	})
	logOp(c, "reorder-pages", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Crop Pages
// ──────────────────────────────────────────────────────────

func CropPagesHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	box := strings.TrimSpace(c.DefaultPostForm("box", ""))
	if box == "" {
		sendError(c, 400, "Specify crop box, e.g. '[0 0 400 600]'")
		return
	}
	pages := splitPages(c.DefaultPostForm("pages", ""))

	safeStream(c, "application/pdf", "cropped.pdf", func(w io.Writer) error {
		return services.CropPages(handle, w, pages, box)
	})
	logOp(c, "crop", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Insert Blank Pages
// ──────────────────────────────────────────────────────────

func InsertBlankHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	afterPages := splitPages(c.DefaultPostForm("after", ""))
	beforeFlag := c.DefaultPostForm("mode", "after") == "before"

	safeStream(c, "application/pdf", "with_blanks.pdf", func(w io.Writer) error {
		return services.InsertBlankPages(handle, w, afterPages, beforeFlag)
	})
	logOp(c, "insert-blank", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Add Page Numbers
// ──────────────────────────────────────────────────────────

func AddPageNumbersHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	position := c.DefaultPostForm("position", "bottom-center")
	startNum := 1

	safeStream(c, "application/pdf", "numbered.pdf", func(w io.Writer) error {
		return services.AddPageNumbers(handle, w, position, startNum)
	})
	logOp(c, "page-numbers", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Extract Text
// ──────────────────────────────────────────────────────────

func ExtractTextHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	pages, err := services.ExtractText(handle)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": handle.Name,
		"pages":    pages,
		"count":    len(pages),
	})
	logOp(c, "extract-text", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Extract Images
// ──────────────────────────────────────────────────────────

func ExtractImagesHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	images, err := services.ExtractImages(handle)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}
	if len(images) == 0 {
		sendError(c, 422, "No embedded images found")
		return
	}

	safeStream(c, "application/zip", "extracted_images.zip", func(w io.Writer) error {
		return services.ZipImageData(images, w)
	})
	logOp(c, "extract-images", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Page Count (JSON response)
// ──────────────────────────────────────────────────────────

func PageCountHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	count, err := services.GetPageCount(handle)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}
	c.JSON(200, gin.H{"filename": handle.Name, "pages": count})
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
