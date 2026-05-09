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

func SplitPDF(inFile, outDir string) ([]string, error) {
	if err := api.SplitFile(inFile, outDir, 1, conf()); err != nil {
		return nil, err
	}

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

func CompressPDF(inFile, outFile string) error {
	c := conf()
	c.WriteObjectStream = true
	c.WriteXRefStream = true
	c.Optimize=true
	return api.OptimizeFile(inFile, outFile, c)
}

func RotatePDF(inFile, outFile string, degrees int) error {
	return api.RotateFile(inFile, outFile, degrees, nil, conf())
}

func WatermarkPDF(inFile, outFile, text string, opacity float64, fontSize int) error {
	desc := fmt.Sprintf(
		// "font:Helvetica, points:%d, scale:0.9 rel, color:#808080, opacity:%.2f, rot:45, diagonal:2",
		"font:Helvetica, points:%d, scale:0.9 rel, color:#808080, opacity:%.2f, rot:45",
		fontSize, opacity,
	)
	wm, err := api.TextWatermark(text, desc, false, false, types.POINTS)
	if err != nil {
		return err
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
