package services

import (
	archivezip "archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ──────────────────────────────────────────────────────────
//  Compare two PDFs — metadata + page count/dimensions diff
// ──────────────────────────────────────────────────────────

func ComparePDFs(rs1, rs2 io.ReadSeeker) (*CompareResult, error) {
	info1, err := api.PDFInfo(rs1, "file1", nil, false, conf())
	if err != nil {
		return nil, fmt.Errorf("reading first PDF: %w", err)
	}
	if _, err := rs2.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	info2, err := api.PDFInfo(rs2, "file2", nil, false, conf())
	if err != nil {
		return nil, fmt.Errorf("reading second PDF: %w", err)
	}

	result := &CompareResult{
		File1Pages:  info1.PageCount,
		File2Pages:  info2.PageCount,
		PagesMatch:  info1.PageCount == info2.PageCount,
		File1Title:  info1.Title,
		File2Title:  info2.Title,
		File1Author: info1.Author,
		File2Author: info2.Author,
	}

	// Compare page dimensions
	if _, err := rs1.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if _, err := rs2.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	dims1, _ := api.PageDims(rs1, conf())
	dims2, _ := api.PageDims(rs2, conf())

	minPages := info1.PageCount
	if info2.PageCount < minPages {
		minPages = info2.PageCount
	}
	for i := 0; i < minPages && i < len(dims1) && i < len(dims2); i++ {
		if dims1[i].Width != dims2[i].Width || dims1[i].Height != dims2[i].Height {
			result.PageDiffs = append(result.PageDiffs, PageDiffEntry{
				Page:       i + 1,
				Difference: fmt.Sprintf("Size: %.0fx%.0f vs %.0fx%.0f", dims1[i].Width, dims1[i].Height, dims2[i].Width, dims2[i].Height),
			})
		}
	}

	// Summary
	diffs := len(result.PageDiffs)
	if result.PagesMatch && diffs == 0 {
		result.Summary = "PDFs have identical page count and dimensions"
	} else {
		parts := []string{}
		if !result.PagesMatch {
			parts = append(parts, fmt.Sprintf("Page count differs: %d vs %d", info1.PageCount, info2.PageCount))
		}
		if diffs > 0 {
			parts = append(parts, fmt.Sprintf("%d pages have different dimensions", diffs))
		}
		result.Summary = strings.Join(parts, "; ")
	}

	return result, nil
}

// ──────────────────────────────────────────────────────────
//  Repair PDF — read + validate + optimize with relaxed mode
// ──────────────────────────────────────────────────────────

func RepairPDF(rs io.ReadSeeker, w io.Writer) error {
	c := conf()
	c.ValidationMode = 0 // ValidationRelaxed
	c.WriteObjectStream = true
	c.WriteXRefStream = true
	return api.Optimize(rs, w, c)
}

// ──────────────────────────────────────────────────────────
//  JPG → PDF
// ──────────────────────────────────────────────────────────

func JPGToPDF(imageReaders []io.Reader, w io.Writer) error {
	imp, err := api.Import("dpi:72, pos:full, sc:1.0 abs", types.POINTS)
	if err != nil {
		return err
	}
	return api.ImportImages(nil, w, imageReaders, imp, conf())
}

// JPGToPDFZip converts each image to an individual PDF and zips them.
func JPGToPDFZip(imageReaders []io.Reader, names []string, w io.Writer) error {
	imp, err := api.Import("dpi:72, pos:full, sc:1.0 abs", types.POINTS)
	if err != nil {
		return err
	}

	zw := archivezip.NewWriter(w)
	defer zw.Close()

	for i, rdr := range imageReaders {
		var buf bytes.Buffer
		if err := api.ImportImages(nil, &buf, []io.Reader{rdr}, imp, conf()); err != nil {
			return fmt.Errorf("converting image %d: %w", i+1, err)
		}
		name := fmt.Sprintf("image_%d.pdf", i+1)
		if i < len(names) {
			base := strings.TrimSuffix(names[i], ".jpg")
			base = strings.TrimSuffix(base, ".jpeg")
			base = strings.TrimSuffix(base, ".png")
			base = strings.TrimSuffix(base, ".gif")
			base = strings.TrimSuffix(base, ".tiff")
			base = strings.TrimSuffix(base, ".webp")
			name = base + ".pdf"
		}
		fw, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := fw.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────
//  PDF → JPG
// ──────────────────────────────────────────────────────────

func PDFToJPG(rs io.ReadSeeker) ([]ImageData, error) {
	imagesPerPage, err := api.ExtractImagesRaw(rs, nil, conf())
	if err != nil {
		if !strings.Contains(err.Error(), "no images") {
			return nil, err
		}
	}

	var out []ImageData
	for _, pageImages := range imagesPerPage {
		for _, img := range pageImages {
			data, err := io.ReadAll(img.Reader)
			if err != nil {
				continue
			}
			name := fmt.Sprintf("page_%d_%s.%s", img.PageNr, img.Name, img.FileType)
			if img.FileType != "jpg" && img.FileType != "jpeg" {
				jpgData, jpgErr := convertBytesToJPEG(data)
				if jpgErr == nil {
					data = jpgData
					name = fmt.Sprintf("page_%d_%s.jpg", img.PageNr, img.Name)
				}
			}
			out = append(out, ImageData{Name: name, Data: data})
		}
	}
	return out, nil
}

func convertBytesToJPEG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Over)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ──────────────────────────────────────────────────────────
//  PDF Info
// ──────────────────────────────────────────────────────────

func GetPDFInfo(rs io.ReadSeeker, filename string) (*pdfcpu.PDFInfo, error) {
	return api.PDFInfo(rs, filename, nil, false, conf())
}
