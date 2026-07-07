package controllers

import (
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  Password Protect Files — accepts ANY file types (PDFs,
//  images, documents, etc.) and wraps them in an AES-256
//  encrypted ZIP archive.
// ──────────────────────────────────────────────────────────

func ProtectFilesHandler(c *gin.Context) {
	password := strings.TrimSpace(c.DefaultPostForm("password", ""))
	if password == "" {
		sendError(c, 400, "Password is required")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		sendError(c, 400, "Could not parse form: "+err.Error())
		return
	}

	fhs := form.File["files"]
	if len(fhs) == 0 {
		sendError(c, 400, "Upload at least one file to protect")
		return
	}

	var files []services.NamedBuffer
	for _, fh := range fhs {
		f, err := fh.Open()
		if err != nil {
			sendError(c, 400, fmt.Sprintf("Could not open %s: %v", fh.Filename, err))
			return
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			sendError(c, 400, fmt.Sprintf("Could not read %s: %v", fh.Filename, err))
			return
		}
		files = append(files, services.NamedBuffer{Name: fh.Filename, Data: data})
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="protected.zip"`)
	c.Status(200)

	if err := services.EncryptedZipFiles(files, password, c.Writer); err != nil {
		c.Error(err)
		return
	}

	if len(fhs) > 0 {
		logOp(c, "protect-files", fhs[0].Filename)
	}
}
