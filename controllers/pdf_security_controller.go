package controllers

import (
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  Encrypt PDF
// ──────────────────────────────────────────────────────────

func EncryptPDFHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	userPW := strings.TrimSpace(c.DefaultPostForm("password", ""))
	ownerPW := strings.TrimSpace(c.DefaultPostForm("owner_password", ""))
	if userPW == "" {
		sendError(c, 400, "Password is required")
		return
	}

	safeStream(c, "application/pdf", "encrypted.pdf", func(w io.Writer) error {
		return services.EncryptPDF(handle, w, userPW, ownerPW)
	})
	logOp(c, "encrypt", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Decrypt PDF
// ──────────────────────────────────────────────────────────

func DecryptPDFHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	password := strings.TrimSpace(c.DefaultPostForm("password", ""))
	if password == "" {
		sendError(c, 400, "Password is required to decrypt")
		return
	}

	safeStream(c, "application/pdf", "decrypted.pdf", func(w io.Writer) error {
		return services.DecryptPDF(handle, w, password)
	})
	logOp(c, "decrypt", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Redact
// ──────────────────────────────────────────────────────────

func RedactPDFHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	areasRaw := strings.TrimSpace(c.DefaultPostForm("areas", ""))
	if areasRaw == "" {
		sendError(c, 400, "Specify areas to redact, e.g. '1:50,700,300,750'")
		return
	}
	areas := strings.Split(areasRaw, ";")

	safeStream(c, "application/pdf", "redacted.pdf", func(w io.Writer) error {
		return services.RedactAreas(handle, w, areas)
	})
	logOp(c, "redact", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Sign PDF (visual stamp)
// ──────────────────────────────────────────────────────────

func SignPDFHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	signText := c.DefaultPostForm("text", "Signed by User")
	pages := splitPages(c.DefaultPostForm("pages", ""))
	if len(pages) == 0 {
		// Sign last page by default
		count, countErr := services.GetPageCount(handle)
		if countErr == nil && count > 0 {
			pages = []string{strconv.Itoa(count)}
		}
		handle.Seek(0, 0)
	}

	safeStream(c, "application/pdf", "signed.pdf", func(w io.Writer) error {
		return services.VisualSignPDF(handle, w, signText, pages)
	})
	logOp(c, "sign", handle.Name)
}
