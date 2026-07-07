package services

import (
	"fmt"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ──────────────────────────────────────────────────────────
//  Delete Pages
// ──────────────────────────────────────────────────────────

func DeletePages(rs io.ReadSeeker, w io.Writer, pages []string) error {
	return api.RemovePages(rs, w, pages, conf())
}

// ──────────────────────────────────────────────────────────
//  Reorder Pages — uses Collect to produce pages in given order
// ──────────────────────────────────────────────────────────

func ReorderPages(rs io.ReadSeeker, w io.Writer, pageOrder []string) error {
	return api.Collect(rs, w, pageOrder, conf())
}

// ──────────────────────────────────────────────────────────
//  Crop Pages
// ──────────────────────────────────────────────────────────

func CropPages(rs io.ReadSeeker, w io.Writer, pages []string, boxStr string) error {
	// boxStr format: "[0 0 400 600]" or dimensions like "200 300"
	box, err := model.ParseBox(boxStr, types.POINTS)
	if err != nil {
		return fmt.Errorf("invalid crop box %q: %w", boxStr, err)
	}
	return api.Crop(rs, w, pages, box, conf())
}

// ──────────────────────────────────────────────────────────
//  Insert Blank Pages
// ──────────────────────────────────────────────────────────

func InsertBlankPages(rs io.ReadSeeker, w io.Writer, afterPages []string, before bool) error {
	// Default A4 page config
	pageConf := pdfcpu.DefaultPageConfiguration()
	return api.InsertPages(rs, w, afterPages, before, pageConf, conf())
}

// ──────────────────────────────────────────────────────────
//  Add Page Numbers — watermark stamp with %p variable
// ──────────────────────────────────────────────────────────

func AddPageNumbers(rs io.ReadSeeker, w io.Writer, position string, startNum int) error {
	if position == "" {
		position = "bc" // bottom-center default
	}

	// Map friendly position names to pdfcpu anchor positions
	posMap := map[string]string{
		"top-left": "tl", "top-center": "tc", "top-right": "tr",
		"bottom-left": "bl", "bottom-center": "bc", "bottom-right": "br",
		"tl": "tl", "tc": "tc", "tr": "tr",
		"bl": "bl", "bc": "bc", "br": "br",
	}
	anchor, ok := posMap[strings.ToLower(position)]
	if !ok {
		anchor = "bc"
	}

	desc := fmt.Sprintf("font:Helvetica, points:10, pos:%s, offset:0 5, fillc:#333333, opacity:0.8", anchor)
	pageNumText := fmt.Sprintf("Page %%p")
	if startNum > 1 {
		pageNumText = fmt.Sprintf("Page %%p") // pdfcpu uses %p for page number
	}

	wm, err := api.TextWatermark(pageNumText, desc, true, true, types.POINTS)
	if err != nil {
		return fmt.Errorf("page numbers config: %w", err)
	}
	return api.AddWatermarks(rs, w, nil, wm, conf())
}

// ──────────────────────────────────────────────────────────
//  Extract Text — extracts content streams as text per page
// ──────────────────────────────────────────────────────────

type PageText struct {
	Page int    `json:"page"`
	Text string `json:"text"`
}

func ExtractText(rs io.ReadSeeker) ([]PageText, error) {
	c := conf()
	c.Cmd = model.EXTRACTCONTENT

	ctx, err := api.ReadValidateAndOptimize(rs, c)
	if err != nil {
		return nil, err
	}

	pages, err := api.PagesForPageSelection(ctx.PageCount, nil, true, true)
	if err != nil {
		return nil, err
	}

	var results []PageText
	for p, v := range pages {
		if !v {
			continue
		}
		r, err := pdfcpu.ExtractPageContent(ctx, p)
		if err != nil {
			continue
		}
		if r == nil {
			continue
		}
		data, err := io.ReadAll(r)
		if err != nil {
			continue
		}
		text := string(data)
		if strings.TrimSpace(text) != "" {
			results = append(results, PageText{Page: p, Text: text})
		}
	}

	return results, nil
}

// ──────────────────────────────────────────────────────────
//  Extract Images — re-exported from ExtractImagesRaw
// ──────────────────────────────────────────────────────────

func ExtractImages(rs io.ReadSeeker) ([]ImageData, error) {
	return PDFToJPG(rs) // reuse existing logic
}

// ──────────────────────────────────────────────────────────
//  Page Count
// ──────────────────────────────────────────────────────────

func GetPageCount(rs io.ReadSeeker) (int, error) {
	return api.PageCount(rs, conf())
}

// ──────────────────────────────────────────────────────────
//  Page Dimensions
// ──────────────────────────────────────────────────────────

type PageDimInfo struct {
	Page   int     `json:"page"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func GetPageDimensions(rs io.ReadSeeker) ([]PageDimInfo, error) {
	dims, err := api.PageDims(rs, conf())
	if err != nil {
		return nil, err
	}
	var out []PageDimInfo
	for i, d := range dims {
		out = append(out, PageDimInfo{Page: i + 1, Width: d.Width, Height: d.Height})
	}
	return out, nil
}
