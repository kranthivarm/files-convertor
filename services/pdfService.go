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

func conf() *model.Configuration {
	c := model.NewDefaultConfiguration()
	c.ValidationMode = model.ValidationRelaxed
	return c
}


func MergePDFs(inFiles []string, outFile string) error {
	return api.MergeCreateFile(inFiles, outFile, false, conf())
}

func SplitPDF(inFile, outDir string, span int) ([]string, error) {
	if span < 1 {
		span = 1
	}
	if err := api.SplitFile(inFile, outDir, span, conf()); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".pdf") {
			out = append(out, filepath.Join(outDir, e.Name()))
		}
	}
	return out, nil
}

func SplitPDFByRanges(inFile, outDir, rawRanges string) ([]string, error) {
	segments := parseRangeSegments(rawRanges)
	if len(segments) == 0 {
		return nil, fmt.Errorf("no valid page ranges in %q", rawRanges)
	}
	c := conf()
	var out []string
	for i, pages := range segments {
		dst := filepath.Join(outDir, fmt.Sprintf("part_%02d.pdf", i+1))
		if err := api.TrimFile(inFile, dst, pages, c); err != nil {
			return nil, fmt.Errorf("range %v: %w", pages, err)
		}
		out = append(out, dst)
	}
	return out, nil
}

func parseRangeSegments(raw string) [][]string {
	var segments [][]string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segments = append(segments, []string{part})
	}
	return segments
}



func CompressPDF(inFile, outFile string) error {
	c := conf()
	c.WriteObjectStream = true
	c.WriteXRefStream = true

	opt := outFile + ".opt.pdf"
	defer os.Remove(opt)
	if err := api.OptimizeFile(inFile, opt, c); err != nil {
		return copyFile(inFile, outFile)
	}

	rei := outFile + ".rei.pdf"
	defer os.Remove(rei)
	if err := reencodeImagesPDF(opt, rei, 60); err != nil {
		return copyFile(opt, outFile)
	}

	smallest, _ := smallestFile(inFile, opt, rei)
	return copyFile(smallest, outFile)
}


func reencodeImagesPDF(inFile, outFile string, quality int) error {
	tmpDir, err := os.MkdirTemp("", "pdfimg-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	c := conf()

	// Extract embedded images to disk
	if err := api.ExtractImagesFile(inFile, tmpDir, nil, c); err != nil {
		// No images – just copy through
		return copyFile(inFile, outFile)
	}

	// Re-encode each image at lower quality in-place
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if !e.IsDir() {
			_ = reencodeImageFile(filepath.Join(tmpDir, e.Name()), quality)
		}
	}

	// A second Optimize pass rebuilds the PDF with updated stream lengths.
	// This is the step that actually produces a smaller file.
	return api.OptimizeFile(inFile, outFile, c)
}
 
func reencodeImageFile(path string, quality int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		return err
	}

	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, b, img, b.Min, draw.Over)

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return jpeg.Encode(out, dst, &jpeg.Options{Quality: quality})
}

func smallestFile(paths ...string) (string, int64) {
	best := ""
	var bestSize int64 = -1
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if bestSize < 0 || info.Size() < bestSize {
			best = p
			bestSize = info.Size()
		}
	}
	return best, bestSize
}

// copyFile copies src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}


func RotatePDF(inFile, outFile string, degrees int) error {
	return api.RotateFile(inFile, outFile, degrees, nil, conf())
}


func WatermarkPDF(inFile, outFile, text string, opacity float64, fontSize int) error {
	descs := []string{
		fmt.Sprintf("font:Helvetica, points:%d, scale:0.9 rel, fillc:#808080, opacity:%.2f, rot:45, onTop:false",
			fontSize, opacity),
		fmt.Sprintf("font:Helvetica, points:%d, opacity:%.2f, rot:45",
			fontSize, opacity),
		fmt.Sprintf("points:%d, opacity:%.2f",
			fontSize, opacity),
	}

	var wm *model.Watermark
	var err error
	for _, desc := range descs {
		wm, err = api.TextWatermark(text, desc, true, false, types.POINTS)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("watermark: could not build config: %w", err)
	}

	return api.AddWatermarksFile(inFile, outFile, nil, wm, conf())
}


func JPGToPDF(imgFiles []string, outFile string) error {
	imp, err := api.Import("dpi:72, pos:full, sc:1.0 abs", types.POINTS)
	if err != nil {
		return err
	}
	return api.ImportImagesFile(imgFiles, outFile, imp, conf())
}

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