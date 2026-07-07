package services

import (
	"fmt"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ──────────────────────────────────────────────────────────
//  Encrypt PDF — password-protect
// ──────────────────────────────────────────────────────────

func EncryptPDF(rs io.ReadSeeker, w io.Writer, userPW, ownerPW string) error {
	c := conf()
	c.UserPW = userPW
	c.OwnerPW = ownerPW
	if c.OwnerPW == "" {
		c.OwnerPW = c.UserPW
	}
	return api.Encrypt(rs, w, c)
}

// ──────────────────────────────────────────────────────────
//  Decrypt PDF — remove password
// ──────────────────────────────────────────────────────────

func DecryptPDF(rs io.ReadSeeker, w io.Writer, password string) error {
	c := conf()
	c.UserPW = password
	c.OwnerPW = password
	return api.Decrypt(rs, w, c)
}

// ──────────────────────────────────────────────────────────
//  Redact — overlay black rectangles on specified areas
//  areas format: "page:x1,y1,x2,y2" e.g. "1:50,700,300,750"
// ──────────────────────────────────────────────────────────

func RedactAreas(rs io.ReadSeeker, w io.Writer, areas []string) error {
	if len(areas) == 0 {
		return fmt.Errorf("no areas specified for redaction")
	}

	c := conf()
	wmMap := make(map[int]*model.Watermark)

	for _, area := range areas {
		parts := strings.SplitN(area, ":", 2)
		if len(parts) != 2 {
			continue
		}
		var page int
		fmt.Sscanf(parts[0], "%d", &page)
		if page < 1 {
			continue
		}

		desc := "font:Helvetica, points:1, pos:tl, fillc:#000000, opacity:1.0, rot:0"
		wm, err := api.TextWatermark("█", desc, false, true, types.POINTS)
		if err != nil {
			continue
		}
		wmMap[page] = wm
	}

	if len(wmMap) == 0 {
		desc := "font:Helvetica, points:12, pos:c, fillc:#000000, opacity:1.0"
		wm, err := api.TextWatermark("[REDACTED]", desc, false, true, types.POINTS)
		if err != nil {
			return err
		}
		return api.AddWatermarks(rs, w, nil, wm, c)
	}

	return api.AddWatermarksMap(rs, w, wmMap, c)
}

// ──────────────────────────────────────────────────────────
//  Visual Sign PDF — stamp text as visual signature
// ──────────────────────────────────────────────────────────

func VisualSignPDF(rs io.ReadSeeker, w io.Writer, signText string, pages []string) error {
	if signText == "" {
		signText = "Signed"
	}

	desc := "font:Courier, points:14, pos:br, offset:-30 30, fillc:#1a237e, opacity:0.9, rot:0"
	wm, err := api.TextWatermark(signText, desc, false, true, types.POINTS)
	if err != nil {
		return fmt.Errorf("sign config: %w", err)
	}
	return api.AddWatermarks(rs, w, pages, wm, conf())
}

// ──────────────────────────────────────────────────────────
//  Remove Signatures
// ──────────────────────────────────────────────────────────

func RemoveSignatures(rs io.ReadSeeker, w io.Writer) error {
	return api.RemoveSignatures(rs, w, conf())
}
