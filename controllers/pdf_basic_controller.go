package controllers

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/db"
	"i-lov-pdf/middleware"
	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  FileHandle — wraps a multipart.File with a cleanup func.
//  multipart.File already implements io.ReadSeeker; we avoid
//  io.ReadAll entirely, eliminating double-buffering.
// ──────────────────────────────────────────────────────────

type FileHandle struct {
	io.ReadSeeker
	Name    string
	closers []io.Closer
}

func (fh *FileHandle) Close() {
	for _, c := range fh.closers {
		c.Close()
	}
}

// ──────────────────────────────────────────────────────────
//  Helpers — read uploads WITHOUT io.ReadAll
//  multipart.File implements io.ReadSeeker:
//    - files ≤ MaxMultipartMemory → backed by RAM (already in memory)
//    - files > MaxMultipartMemory → backed by OS temp file (kernel-managed)
//  Either way, NO extra copy into a []byte slice.
// ──────────────────────────────────────────────────────────

// readOne returns the multipart file directly as an io.ReadSeeker.
// Caller MUST call handle.Close() when done (use defer).
func readOne(c *gin.Context, field string) (*FileHandle, error) {
	fh, err := c.FormFile(field)
	if err != nil {
		return nil, fmt.Errorf("field %q missing: %w", field, err)
	}
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	// multipart.File implements io.Reader, io.ReaderAt, io.Seeker, io.Closer
	return &FileHandle{
		ReadSeeker: f.(io.ReadSeeker),
		Name:       fh.Filename,
		closers:    []io.Closer{f},
	}, nil
}

// readMany returns multiple multipart files as io.ReadSeekers.
// Caller MUST call Close() on each handle when done.
func readMany(c *gin.Context, field string) ([]*FileHandle, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}
	fhs := form.File[field]
	if len(fhs) == 0 {
		return nil, fmt.Errorf("no files uploaded for field %q", field)
	}
	handles := make([]*FileHandle, 0, len(fhs))
	for _, fh := range fhs {
		f, err := fh.Open()
		if err != nil {
			// Close already-opened handles on error
			for _, h := range handles {
				h.Close()
			}
			return nil, err
		}
		handles = append(handles, &FileHandle{
			ReadSeeker: f.(io.ReadSeeker),
			Name:       fh.Filename,
			closers:    []io.Closer{f},
		})
	}
	return handles, nil
}

// readManyAsReaders returns multiple multipart files as []io.Reader.
// Returns a cleanup function that closes all file handles.
func readManyAsReaders(c *gin.Context, field string) ([]io.Reader, []string, func(), error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, nil, nil, err
	}
	fhs := form.File[field]
	if len(fhs) == 0 {
		return nil, nil, nil, fmt.Errorf("no files uploaded for field %q", field)
	}
	readers := make([]io.Reader, 0, len(fhs))
	names := make([]string, 0, len(fhs))
	var closers []io.Closer
	for _, fh := range fhs {
		f, err := fh.Open()
		if err != nil {
			for _, c := range closers {
				c.Close()
			}
			return nil, nil, nil, err
		}
		readers = append(readers, f)
		names = append(names, fh.Filename)
		closers = append(closers, f)
	}
	cleanup := func() {
		for _, c := range closers {
			c.Close()
		}
	}
	return readers, names, cleanup, nil
}

// readOneFromHeader reads a file from a multipart.FileHeader directly.
// Caller MUST call handle.Close() when done.
func readOneFromHeader(fh *multipart.FileHeader) (*FileHandle, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	return &FileHandle{
		ReadSeeker: f.(io.ReadSeeker),
		Name:       fh.Filename,
		closers:    []io.Closer{f},
	}, nil
}

// ──────────────────────────────────────────────────────────
//  Buffered output — prevent corrupt 200 OK on mid-stream errors
//
//  Instead of writing directly to c.Writer (which flushes headers
//  immediately), we buffer into a pooled bytes.Buffer. Only if the
//  entire operation succeeds do we write headers + body.
// ──────────────────────────────────────────────────────────

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// safeStream buffers the output of fn(). On success, sends headers + body.
// On failure, sends a proper JSON error. No more corrupt truncated downloads.
func safeStream(c *gin.Context, contentType, filename string, fn func(w io.Writer) error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if err := fn(buf); err != nil {
		sendError(c, 500, err.Error())
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Length", strconv.Itoa(buf.Len()))
	c.Status(200)
	c.Writer.Write(buf.Bytes())
}

// log records the operation for authenticated users. No-op for guests.
func logOp(c *gin.Context, operation, filename string) {
	db.LogActivity(middleware.GetUserID(c), operation, filename)
}

// sendError writes a JSON error response.
func sendError(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

// ──────────────────────────────────────────────────────────
//  Handlers
// ──────────────────────────────────────────────────────────

func Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func Merge(c *gin.Context) {
	handles, err := readMany(c, "files")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer func() {
		for _, h := range handles {
			h.Close()
		}
	}()

	if len(handles) < 2 {
		sendError(c, 400, "Upload at least 2 PDFs to merge")
		return
	}

	seekers := make([]io.ReadSeeker, len(handles))
	for i, h := range handles {
		seekers[i] = h.ReadSeeker
	}

	safeStream(c, "application/pdf", "merged.pdf", func(w io.Writer) error {
		return services.MergePDFs(seekers, w)
	})
	logOp(c, "merge", handles[0].Name)
}

func Split(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	rawRanges := strings.TrimSpace(c.DefaultPostForm("ranges", ""))
	var parts []services.NamedBuffer

	if rawRanges != "" {
		parts, err = services.SplitPDFByRanges(handle, rawRanges)
	} else {
		span, _ := strconv.Atoi(c.DefaultPostForm("span", "1"))
		if span < 1 {
			span = 1
		}
		parts, err = services.SplitPDF(handle, span)
	}

	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	safeStream(c, "application/zip", "split_pages.zip", func(w io.Writer) error {
		return services.ZipNamedBuffers(parts, w)
	})
	logOp(c, "split", handle.Name)
}

func Compress(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	var buf bytes.Buffer
	origSize, newSize, err := services.CompressPDF(handle, &buf)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	var savings int
	if origSize > 0 {
		savings = int(100 - newSize*100/origSize)
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="compressed.pdf"`)
	c.Header("X-Original-Size", strconv.FormatInt(origSize, 10))
	c.Header("X-Compressed-Size", strconv.FormatInt(newSize, 10))
	c.Header("X-Savings-Percent", strconv.Itoa(savings))
	c.Header("Content-Length", strconv.Itoa(buf.Len()))
	c.Data(200, "application/pdf", buf.Bytes())
	logOp(c, "compress", handle.Name)
}

func Rotate(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	degrees, _ := strconv.Atoi(c.DefaultPostForm("degrees", "90"))

	safeStream(c, "application/pdf", "rotated.pdf", func(w io.Writer) error {
		return services.RotatePDF(handle, w, degrees)
	})
	logOp(c, "rotate", handle.Name)
}

func Watermark(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	text := c.DefaultPostForm("text", "CONFIDENTIAL")
	opacity, _ := strconv.ParseFloat(c.DefaultPostForm("opacity", "0.3"), 64)
	fontSize, _ := strconv.Atoi(c.DefaultPostForm("fontsize", "48"))

	safeStream(c, "application/pdf", "watermarked.pdf", func(w io.Writer) error {
		return services.WatermarkPDF(handle, w, text, opacity, fontSize)
	})
	logOp(c, "watermark", handle.Name)
}

func JPGToPDF(c *gin.Context) {
	imgReaders, names, cleanup, err := readManyAsReaders(c, "files")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer cleanup()

	mode := c.PostForm("mode") // "merged" (default) or "zip"

	if mode == "zip" {
		safeStream(c, "application/zip", "images_as_pdfs.zip", func(w io.Writer) error {
			return services.JPGToPDFZip(imgReaders, names, w)
		})
	} else {
		safeStream(c, "application/pdf", "images.pdf", func(w io.Writer) error {
			return services.JPGToPDF(imgReaders, w)
		})
	}
	logOp(c, "jpg-to-pdf", names[0])
}

func PDFToJPG(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	images, err := services.PDFToJPG(handle)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}
	if len(images) == 0 {
		sendError(c, 422, "No embedded images found. Text-only PDFs need a render engine (pdftoppm).")
		return
	}

	safeStream(c, "application/zip", "pages.zip", func(w io.Writer) error {
		return services.ZipImageData(images, w)
	})
	logOp(c, "pdf-to-jpg", handle.Name)
}