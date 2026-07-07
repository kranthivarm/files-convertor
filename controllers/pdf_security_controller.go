package controllers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  Encrypt PDF
// ──────────────────────────────────────────────────────────

func EncryptPDFHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	userPW := strings.TrimSpace(c.DefaultPostForm("password", ""))
	ownerPW := strings.TrimSpace(c.DefaultPostForm("owner_password", ""))
	if userPW == "" {
		sendError(c, 400, "Password is required")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="encrypted.pdf"`)
	c.Status(200)
	if err := services.EncryptPDF(rs, c.Writer, userPW, ownerPW); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "encrypt", origName)
}

// ──────────────────────────────────────────────────────────
//  Decrypt PDF
// ──────────────────────────────────────────────────────────

func DecryptPDFHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	password := strings.TrimSpace(c.DefaultPostForm("password", ""))
	if password == "" {
		sendError(c, 400, "Password is required to decrypt")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="decrypted.pdf"`)
	c.Status(200)
	if err := services.DecryptPDF(rs, c.Writer, password); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "decrypt", origName)
}

// ──────────────────────────────────────────────────────────
//  Redact
// ──────────────────────────────────────────────────────────

func RedactPDFHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	areasRaw := strings.TrimSpace(c.DefaultPostForm("areas", ""))
	if areasRaw == "" {
		sendError(c, 400, "Specify areas to redact, e.g. '1:50,700,300,750'")
		return
	}
	areas := strings.Split(areasRaw, ";")

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="redacted.pdf"`)
	c.Status(200)
	if err := services.RedactAreas(rs, c.Writer, areas); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "redact", origName)
}

// ──────────────────────────────────────────────────────────
//  Sign PDF (visual stamp)
// ──────────────────────────────────────────────────────────

func SignPDFHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	signText := c.DefaultPostForm("text", "Signed by User")
	pages := splitPages(c.DefaultPostForm("pages", ""))
	if len(pages) == 0 {
		// Sign last page by default
		count, countErr := services.GetPageCount(rs)
		if countErr == nil && count > 0 {
			pages = []string{strconv.Itoa(count)}
		}
		rs.Seek(0, 0)
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="signed.pdf"`)
	c.Status(200)
	if err := services.VisualSignPDF(rs, c.Writer, signText, pages); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "sign", origName)
}
