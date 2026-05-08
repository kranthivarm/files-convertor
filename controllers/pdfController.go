package controllers

import (
	"fmt"
	"i-lov-pdf/services"
	"i-lov-pdf/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

func Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func Merge(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	files := form.File["files"]
	if len(files) < 2 {
		c.JSON(400, gin.H{"error": "need at least 2 PDFs"})
		return
	}

	var paths []string
	for _, fh := range files {
		dst := filepath.Join("uploads", utils.Uid()+"_"+fh.Filename)
		if err := c.SaveUploadedFile(fh, dst); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		paths = append(paths, dst)
	}
	out := filepath.Join("outputs", utils.Uid()+"_merged.pdf")
	if err := services.MergePDFs(paths, out); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": fmt.Sprintf("Merged %d PDFs", len(paths)),
		"download": "/outputs/" + filepath.Base(out), "filename": "merged.pdf"})
}

func Split(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	src := filepath.Join("uploads", utils.Uid()+"_"+fh.Filename)
	c.SaveUploadedFile(fh, src)

	outDir := filepath.Join("outputs", utils.Uid()+"_split")
	os.MkdirAll(outDir, 0755)
	pages, err := services.SplitPDF(src, outDir)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	zip := outDir + ".zip"
	services.ZipFiles(pages, zip)
	c.JSON(200, gin.H{"message": fmt.Sprintf("Split into %d pages", len(pages)),
		"download": "/outputs/" + filepath.Base(zip), "filename": "split_pages.zip", "pages": len(pages)})
}

func Compress(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	src := filepath.Join("uploads", utils.Uid()+"_"+fh.Filename)
	c.SaveUploadedFile(fh, src)
	origInfo, _ := os.Stat(src)

	out := filepath.Join("outputs", utils.Uid()+"_compressed.pdf")
	if err := services.CompressPDF(src, out); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	newInfo, _ := os.Stat(out)
	savings := int(100 - newInfo.Size()*100/origInfo.Size())
	c.JSON(200, gin.H{"message": fmt.Sprintf("Saved ~%d%%", savings),
		"download": "/outputs/" + filepath.Base(out), "filename": "compressed.pdf",
		"originalSize": origInfo.Size(), "newSize": newInfo.Size(), "savings": savings})
}

func Rotate(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	src := filepath.Join("uploads", utils.Uid()+"_"+fh.Filename)
	c.SaveUploadedFile(fh, src)

	degrees, _ := strconv.Atoi(c.DefaultPostForm("degrees", "90"))
	out := filepath.Join("outputs", utils.Uid()+"_rotated.pdf")
	if err := services.RotatePDF(src, out, degrees); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": fmt.Sprintf("Rotated %d°", degrees),
		"download": "/outputs/" + filepath.Base(out), "filename": "rotated.pdf"})
}

func Watermark(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	src := filepath.Join("uploads", utils.Uid()+"_"+fh.Filename)
	c.SaveUploadedFile(fh, src)

	text := c.DefaultPostForm("text", "CONFIDENTIAL")
	opacity, _ := strconv.ParseFloat(c.DefaultPostForm("opacity", "0.3"), 64)
	fontSize, _ := strconv.Atoi(c.DefaultPostForm("fontsize", "48"))

	out := filepath.Join("outputs", utils.Uid()+"_watermarked.pdf")
	if err := services.WatermarkPDF(src, out, text, opacity, fontSize); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": fmt.Sprintf("Watermark \"%s\" applied", text),
		"download": "/outputs/" + filepath.Base(out), "filename": "watermarked.pdf"})
}

func JPGToPDF(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File["files"]
	var paths []string
	for _, fh := range files {
		dst := filepath.Join("uploads", utils.Uid()+"_"+fh.Filename)
		c.SaveUploadedFile(fh, dst)
		paths = append(paths, dst)
	}
	out := filepath.Join("outputs", utils.Uid()+"_images.pdf")
	if err := services.JPGToPDF(paths, out); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": fmt.Sprintf("Converted %d images to PDF", len(paths)),
		"download": "/outputs/" + filepath.Base(out), "filename": "images.pdf"})
}

func PDFToJPG(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	src := filepath.Join("uploads", utils.Uid()+"_"+fh.Filename)
	c.SaveUploadedFile(fh, src)

	dpi, _ := strconv.Atoi(c.DefaultPostForm("dpi", "150"))
	outDir := filepath.Join("outputs", utils.Uid()+"_pages")
	os.MkdirAll(outDir, 0755)

	pages, err := services.PDFToJPG(src, outDir, dpi)
	if err != nil || len(pages) == 0 {
		c.JSON(422, gin.H{"error": "No embedded images found"})
		return
	}
	zip := outDir + ".zip"
	services.ZipFiles(pages, zip)
	c.JSON(200, gin.H{"message": fmt.Sprintf("Extracted %d images", len(pages)),
		"download": "/outputs/" + filepath.Base(zip), "filename": "pages.zip", "pages": len(pages)})
}
