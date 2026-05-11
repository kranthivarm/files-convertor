package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	     "i-lov-pdf/middleware"
	     "i-lov-pdf/db"
	
	     "i-lov-pdf/services"
)


func uid() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

func saveOne(c *gin.Context, field, dir string) (string, string, error) {
	fh, err := c.FormFile(field)
	if err != nil {
		return "", "", fmt.Errorf("field %q missing: %w", field, err)
	}
	os.MkdirAll(dir, 0755)
	dst := filepath.Join(dir, uid()+"_"+filepath.Base(fh.Filename))
	return dst, fh.Filename, c.SaveUploadedFile(fh, dst)
}


func saveMany(c *gin.Context, field, dir string) ([]string, string, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, "", err
	}
	fhs := form.File[field]
	if len(fhs) == 0 {
		return nil, "", fmt.Errorf("no files uploaded for field %q", field)
	}
	os.MkdirAll(dir, 0755)
	var paths []string
	for _, fh := range fhs {
		dst := filepath.Join(dir, uid()+"_"+filepath.Base(fh.Filename))
		if err := c.SaveUploadedFile(fh, dst); err != nil {
			return nil, "", err
		}
		paths = append(paths, dst)
	}
	return paths, fhs[0].Filename, nil
}

// log records the operation for authenticated users. No-op for guests.
func logOp(c *gin.Context, operation, filename string) {
	db.LogActivity(middleware.GetUserID(c), operation, filename)
}


func Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}


func Merge(c *gin.Context) {
	paths, firstName, err := saveMany(c, "files", "uploads")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if len(paths) < 2 {
		c.JSON(400, gin.H{"error": "Upload at least 2 PDFs to merge"})
		return
	}
	out := filepath.Join("outputs", uid()+"_merged.pdf")
	if err := services.MergePDFs(paths, out); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	logOp(c, "merge", firstName)
	c.JSON(200, gin.H{
		"message":  fmt.Sprintf("Merged %d PDFs successfully", len(paths)),
		"download": "/outputs/" + filepath.Base(out),
		"filename": "merged.pdf",
	})
}


func Split(c *gin.Context) {
	src, origName, err := saveOne(c, "file", "uploads")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	outDir := filepath.Join("outputs", uid()+"_split")
	os.MkdirAll(outDir, 0755)

	rawRanges := strings.TrimSpace(c.DefaultPostForm("ranges", ""))
	var pages []string

	if rawRanges != "" {
		pages, err = services.SplitPDFByRanges(src, outDir, rawRanges)
	} else {
		span, _ := strconv.Atoi(c.DefaultPostForm("span", "1"))
		if span < 1 {
			span = 1
		}
		pages, err = services.SplitPDF(src, outDir, span)
	}

	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	zipPath := outDir + ".zip"
	if err := services.ZipFiles(pages, zipPath); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	logOp(c, "split", origName)
	c.JSON(200, gin.H{
		"message":  fmt.Sprintf("Split into %d part(s)", len(pages)),
		"download": "/outputs/" + filepath.Base(zipPath),
		"filename": "split_pages.zip",
		"pages":    len(pages),
	})
}


func Compress(c *gin.Context) {
	src, origName, err := saveOne(c, "file", "uploads")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	origInfo, _ := os.Stat(src)
	out := filepath.Join("outputs", uid()+"_compressed.pdf")

	if err := services.CompressPDF(src, out); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	newInfo, _ := os.Stat(out)

	var savings int
	if origInfo != nil && newInfo != nil && origInfo.Size() > 0 {
		savings = int(100 - newInfo.Size()*100/origInfo.Size())
	}
	logOp(c, "compress", origName)
	c.JSON(200, gin.H{
		"message":      fmt.Sprintf("Compressed — saved ~%d%%", savings),
		"download":     "/outputs/" + filepath.Base(out),
		"filename":     "compressed.pdf",
		"originalSize": origInfo.Size(),
		"newSize":      newInfo.Size(),
		"savings":      savings,
	})
}


func Rotate(c *gin.Context) {
	src, origName, err := saveOne(c, "file", "uploads")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	degrees, _ := strconv.Atoi(c.DefaultPostForm("degrees", "90"))
	out := filepath.Join("outputs", uid()+"_rotated.pdf")

	if err := services.RotatePDF(src, out, degrees); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	logOp(c, "rotate", origName)
	c.JSON(200, gin.H{
		"message":  fmt.Sprintf("Rotated %d°", degrees),
		"download": "/outputs/" + filepath.Base(out),
		"filename": "rotated.pdf",
	})
}


func Watermark(c *gin.Context) {
	src, origName, err := saveOne(c, "file", "uploads")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	text := c.DefaultPostForm("text", "CONFIDENTIAL")
	opacity, _ := strconv.ParseFloat(c.DefaultPostForm("opacity", "0.3"), 64)
	fontSize, _ := strconv.Atoi(c.DefaultPostForm("fontsize", "48"))

	out := filepath.Join("outputs", uid()+"_watermarked.pdf")
	if err := services.WatermarkPDF(src, out, text, opacity, fontSize); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	logOp(c, "watermark", origName)
	c.JSON(200, gin.H{
		"message":  fmt.Sprintf("Watermark \"%s\" applied", text),
		"download": "/outputs/" + filepath.Base(out),
		"filename": "watermarked.pdf",
	})
}


func JPGToPDF(c *gin.Context) {
	paths, firstName, err := saveMany(c, "files", "uploads")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	out := filepath.Join("outputs", uid()+"_images.pdf")
	if err := services.JPGToPDF(paths, out); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	logOp(c, "jpg-to-pdf", firstName)
	c.JSON(200, gin.H{
		"message":  fmt.Sprintf("Converted %d image(s) to PDF", len(paths)),
		"download": "/outputs/" + filepath.Base(out),
		"filename": "images.pdf",
	})
}


func PDFToJPG(c *gin.Context) {
	src, origName, err := saveOne(c, "file", "uploads")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	dpi, _ := strconv.Atoi(c.DefaultPostForm("dpi", "150"))
	outDir := filepath.Join("outputs", uid()+"_pages")
	os.MkdirAll(outDir, 0755)

	pages, err := services.PDFToJPG(src, outDir, dpi)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if len(pages) == 0 {
		c.JSON(422, gin.H{"error": "No embedded images found. Text-only PDFs need a render engine (pdftoppm)."})
		return
	}
	zipPath := outDir + ".zip"
	if err := services.ZipFiles(pages, zipPath); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	logOp(c, "pdf-to-jpg", origName)
	c.JSON(200, gin.H{
		"message":  fmt.Sprintf("Extracted %d image(s) from PDF", len(pages)),
		"download": "/outputs/" + filepath.Base(zipPath),
		"filename": "pages.zip",
		"pages":    len(pages),
	})
}