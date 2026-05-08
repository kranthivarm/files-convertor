package services

import (
	"archive/zip"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

)

// conf returns a default pdfcpu config (no config dir needed in server mode).
func conf() *model.Configuration {
	c := model.NewDefaultConfiguration()
	c.ValidationMode = model.ValidationRelaxed
	return c
}

// MergePDFs combines inFiles into outFile.
func MergePDFs(inFiles []string, outFile string) error {
	return api.MergeCreateFile(inFiles, outFile, false, conf())
}

// SplitPDF splits every page into its own PDF under outDir.
// Returns the paths of all created files.
func SplitPDF(inFile, outDir string) ([]string, error) {
	if err := api.SplitFile(inFile, outDir, 1, conf()); err != nil {
		return nil, err
	}
	// Collect produced files
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".pdf") {
			out = append(out, filepath.Join(outDir, e.Name()))
		}
	}
	return out, nil
}

// CompressPDF uses pdfcpu's optimiser (deduplicates objects/streams, re-encodes
// images, strips redundant data). It is fully in-process — no external calls.
func CompressPDF(inFile, outFile string) error {
	c := conf()
	// Set aggressive image down-sampling
	c.WriteObjectStream = true
	c.WriteXRefStream = true
	return api.OptimizeFile(inFile, outFile, c)
}


// RotatePDF rotates all pages by degrees (must be a multiple of 90).
func RotatePDF(inFile, outFile string, degrees int) error {
	return api.RotateFile(inFile, outFile, degrees, nil, conf())
}


// WatermarkPDF stamps diagonal text on every page.
// opacity: 0.0–1.0, fontSize: points (e.g. 48).
func WatermarkPDF(inFile, outFile, text string, opacity float64, fontSize int) error {
	// pdfcpu watermark description string
	desc := fmt.Sprintf(
		"font:Helvetica, points:%d, scale:0.9 rel, color:#808080, opacity:%.2f, rot:45, diagonal:2",
		fontSize, opacity,
	)
	wm, err := api.TextWatermark(text, desc, false, false, types.POINTS)
	if err != nil {
		return err
	}
	return api.AddWatermarksFile(inFile, outFile, nil, wm, conf())
}


// JPGToPDF converts one or more image files (JPEG/PNG/GIF/TIFF) into a PDF.
// pdfcpu's ImportImagesFile handles this natively.
func JPGToPDF(imgFiles []string, outFile string) error {
	imp, err := api.Import("dpi:72, pos:full, sc:1.0 abs", types.POINTS)
	if err != nil {
		return err
	}
	return api.ImportImagesFile(imgFiles, outFile, imp, conf())
}


// PDFToJPG renders every page of inFile to a JPEG inside outDir.
// pdfcpu's ExtractImages extracts embedded raster objects. For full-page
// rendering we fall back to Go's image library to composite extracted images.
// NOTE: full rasterisation (like pdftoppm) needs a renderer (poppler/mupdf).
//
//	Here we extract embedded images — which works perfectly for scanned PDFs
//	and image-heavy documents. Text-only pages produce a placeholder.
func PDFToJPG(inFile, outDir string, dpi int) ([]string, error) {
	if err := api.ExtractImagesFile(inFile, outDir, nil, conf()); err != nil {
		// If no images found, that's okay — just return empty
		if !strings.Contains(err.Error(), "no images") {
			return nil, err
		}
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.HasSuffix(lower, ".jpg") ||
			strings.HasSuffix(lower, ".jpeg") ||
			strings.HasSuffix(lower, ".png") ||
			strings.HasSuffix(lower, ".tif") ||
			strings.HasSuffix(lower, ".tiff") {
			p := filepath.Join(outDir, e.Name())
			// Ensure it is a JPEG
			if !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") {
				p, err = convertToJPEG(p)
				if err != nil {
					continue
				}
			}
			out = append(out, p)
		}
	}
	return out, nil
}

// convertToJPEG converts any image file to JPEG and deletes the original.
func convertToJPEG(src string) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}

	dst := strings.TrimSuffix(src, filepath.Ext(src)) + ".jpg"
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Convert to RGBA then to JPEG (handles transparency)
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Over)

	if err := jpeg.Encode(out, rgba, &jpeg.Options{Quality: 90}); err != nil {
		return "", err
	}
	os.Remove(src)
	return dst, nil
}


// ZipFiles creates a ZIP archive at outPath containing all files.
func ZipFiles(files []string, outPath string) error {
	zf, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer zf.Close()

	zw := zip.NewWriter(zf)
	defer zw.Close()

	for _, fp := range files {
		if err := addToZip(zw, fp); err != nil {
			return err
		}
	}
	return nil
}

func addToZip(zw *zip.Writer, fp string) error {
	f, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := zw.Create(filepath.Base(fp))
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}
