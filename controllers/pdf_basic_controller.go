package controllers

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/db"
	"i-lov-pdf/middleware"
	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  Helpers — read uploads into memory (no disk writes)
// ──────────────────────────────────────────────────────────

// readOne reads a single uploaded file into a bytes.Reader (implements io.ReadSeeker).
func readOne(c *gin.Context, field string) (*bytes.Reader, string, error) {
	fh, err := c.FormFile(field)
	if err != nil {
		return nil, "", fmt.Errorf("field %q missing: %w", field, err)
	}
	f, err := fh.Open()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(data), fh.Filename, nil
}

// readMany reads multiple uploaded files into bytes.Readers.
func readMany(c *gin.Context, field string) ([]*bytes.Reader, []string, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, nil, err
	}
	fhs := form.File[field]
	if len(fhs) == 0 {
		return nil, nil, fmt.Errorf("no files uploaded for field %q", field)
	}
	var readers []*bytes.Reader
	var names []string
	for _, fh := range fhs {
		f, err := fh.Open()
		if err != nil {
			return nil, nil, err
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, nil, err
		}
		readers = append(readers, bytes.NewReader(data))
		names = append(names, fh.Filename)
	}
	return readers, names, nil
}

// readManyAsReaders reads multiple uploaded files as io.Reader slices (for ImportImages).
func readManyAsReaders(c *gin.Context, field string) ([]io.Reader, []string, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, nil, err
	}
	fhs := form.File[field]
	if len(fhs) == 0 {
		return nil, nil, fmt.Errorf("no files uploaded for field %q", field)
	}
	var readers []io.Reader
	var names []string
	for _, fh := range fhs {
		f, err := fh.Open()
		if err != nil {
			return nil, nil, err
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, nil, err
		}
		readers = append(readers, bytes.NewReader(data))
		names = append(names, fh.Filename)
	}
	return readers, names, nil
}

// log records the operation for authenticated users. No-op for guests.
func logOp(c *gin.Context, operation, filename string) {
	db.LogActivity(middleware.GetUserID(c), operation, filename)
}

// sendError writes a JSON error response. If the client expects a binary response
// (from the new blob-based frontend), we still return JSON for errors.
func sendError(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

// readOneFromHeader reads a file from a multipart.FileHeader into memory.
func readOneFromHeader(fh *multipart.FileHeader) (*bytes.Reader, string, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(data), fh.Filename, nil
}

// ──────────────────────────────────────────────────────────
//  Handlers
// ──────────────────────────────────────────────────────────

func Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func Merge(c *gin.Context) {
	readers, names, err := readMany(c, "files")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	if len(readers) < 2 {
		sendError(c, 400, "Upload at least 2 PDFs to merge")
		return
	}

	// Convert []*bytes.Reader → []io.ReadSeeker
	seekers := make([]io.ReadSeeker, len(readers))
	for i, r := range readers {
		seekers[i] = r
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="merged.pdf"`)
	c.Status(200)

	if err := services.MergePDFs(seekers, c.Writer); err != nil {
		// If we already started writing, we can't send JSON error.
		// Log it; the client will get a truncated response.
		c.Error(err)
		return
	}
	logOp(c, "merge", names[0])
}

func Split(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	rawRanges := strings.TrimSpace(c.DefaultPostForm("ranges", ""))
	var parts []services.NamedBuffer

	if rawRanges != "" {
		parts, err = services.SplitPDFByRanges(rs, rawRanges)
	} else {
		span, _ := strconv.Atoi(c.DefaultPostForm("span", "1"))
		if span < 1 {
			span = 1
		}
		parts, err = services.SplitPDF(rs, span)
	}

	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="split_pages.zip"`)
	c.Status(200)

	if err := services.ZipNamedBuffers(parts, c.Writer); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "split", origName)
}

func Compress(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	var buf bytes.Buffer
	origSize, newSize, err := services.CompressPDF(rs, &buf)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	var savings int
	if origSize > 0 {
		savings = int(100 - newSize*100/origSize)
	}

	// For compress, we return JSON with metadata + base64 would be bad.
	// Instead, we embed the metadata in custom headers and stream the file.
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="compressed.pdf"`)
	c.Header("X-Original-Size", strconv.FormatInt(origSize, 10))
	c.Header("X-Compressed-Size", strconv.FormatInt(newSize, 10))
	c.Header("X-Savings-Percent", strconv.Itoa(savings))
	c.Data(200, "application/pdf", buf.Bytes())
	logOp(c, "compress", origName)
}

func Rotate(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	degrees, _ := strconv.Atoi(c.DefaultPostForm("degrees", "90"))

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="rotated.pdf"`)
	c.Status(200)

	if err := services.RotatePDF(rs, c.Writer, degrees); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "rotate", origName)
}

func Watermark(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	text := c.DefaultPostForm("text", "CONFIDENTIAL")
	opacity, _ := strconv.ParseFloat(c.DefaultPostForm("opacity", "0.3"), 64)
	fontSize, _ := strconv.Atoi(c.DefaultPostForm("fontsize", "48"))

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="watermarked.pdf"`)
	c.Status(200)

	if err := services.WatermarkPDF(rs, c.Writer, text, opacity, fontSize); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "watermark", origName)
}

func JPGToPDF(c *gin.Context) {
	imgReaders, names, err := readManyAsReaders(c, "files")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	mode := c.PostForm("mode") // "merged" (default) or "zip"

	if mode == "zip" {
		// Each image → separate PDF → zip all
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", `attachment; filename="images_as_pdfs.zip"`)
		c.Status(200)

		if err := services.JPGToPDFZip(imgReaders, names, c.Writer); err != nil {
			c.Error(err)
			return
		}
	} else {
		// Default: merge all into one PDF
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", `attachment; filename="images.pdf"`)
		c.Status(200)

		if err := services.JPGToPDF(imgReaders, c.Writer); err != nil {
			c.Error(err)
			return
		}
	}
	logOp(c, "jpg-to-pdf", names[0])
}

func PDFToJPG(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	images, err := services.PDFToJPG(rs)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}
	if len(images) == 0 {
		sendError(c, 422, "No embedded images found. Text-only PDFs need a render engine (pdftoppm).")
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="pages.zip"`)
	c.Status(200)

	if err := services.ZipImageData(images, c.Writer); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "pdf-to-jpg", origName)
}