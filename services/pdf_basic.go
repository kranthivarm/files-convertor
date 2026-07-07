package services

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ──────────────────────────────────────────────────────────
//  Merge
// ──────────────────────────────────────────────────────────

func MergePDFs(readers []io.ReadSeeker, w io.Writer) error {
	return api.MergeRaw(readers, w, false, conf())
}

// ──────────────────────────────────────────────────────────
//  Split
// ──────────────────────────────────────────────────────────

func SplitPDF(rs io.ReadSeeker, span int) ([]NamedBuffer, error) {
	if span < 1 {
		span = 1
	}
	spans, err := api.SplitRaw(rs, span, conf())
	if err != nil {
		return nil, err
	}
	var out []NamedBuffer
	for i, ps := range spans {
		data, err := io.ReadAll(ps.Reader)
		if err != nil {
			return nil, err
		}
		name := fmt.Sprintf("part_%02d_pages_%d-%d.pdf", i+1, ps.From, ps.Thru)
		out = append(out, NamedBuffer{Name: name, Data: data})
	}
	return out, nil
}

func SplitPDFByRanges(rs io.ReadSeeker, rawRanges string) ([]NamedBuffer, error) {
	segments := parseRangeSegments(rawRanges)
	if len(segments) == 0 {
		return nil, fmt.Errorf("no valid page ranges in %q", rawRanges)
	}
	c := conf()
	var out []NamedBuffer
	for i, pages := range segments {
		var buf bytes.Buffer
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		if err := api.Trim(rs, &buf, pages, c); err != nil {
			return nil, fmt.Errorf("range %v: %w", pages, err)
		}
		name := fmt.Sprintf("part_%02d.pdf", i+1)
		out = append(out, NamedBuffer{Name: name, Data: buf.Bytes()})
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

// ──────────────────────────────────────────────────────────
//  Compress
// ──────────────────────────────────────────────────────────

func CompressPDF(rs io.ReadSeeker, w io.Writer) (origSize int64, newSize int64, err error) {
	inputBytes, err := io.ReadAll(rs)
	if err != nil {
		return 0, 0, err
	}
	origSize = int64(len(inputBytes))

	c := conf()
	c.WriteObjectStream = true
	c.WriteXRefStream = true

	reader := bytes.NewReader(inputBytes)
	var buf bytes.Buffer
	if err := api.Optimize(reader, &buf, c); err != nil {
		newSize = origSize
		_, writeErr := w.Write(inputBytes)
		return origSize, newSize, writeErr
	}

	newSize = int64(buf.Len())
	if newSize < origSize {
		_, err = w.Write(buf.Bytes())
	} else {
		newSize = origSize
		_, err = w.Write(inputBytes)
	}
	return origSize, newSize, err
}

// ──────────────────────────────────────────────────────────
//  Rotate
// ──────────────────────────────────────────────────────────

func RotatePDF(rs io.ReadSeeker, w io.Writer, degrees int) error {
	return api.Rotate(rs, w, degrees, nil, conf())
}

// ──────────────────────────────────────────────────────────
//  Watermark
// ──────────────────────────────────────────────────────────

func WatermarkPDF(rs io.ReadSeeker, w io.Writer, text string, opacity float64, fontSize int) error {
	descs := []string{
		fmt.Sprintf("font:Helvetica, points:%d, scale:0.9 rel, fillc:#808080, opacity:%.2f, rot:45, onTop:false",
			fontSize, opacity),
		fmt.Sprintf("font:Helvetica, points:%d, opacity:%.2f, rot:45",
			fontSize, opacity),
		fmt.Sprintf("points:%d, opacity:%.2f",
			fontSize, opacity),
	}

	var wm *model.Watermark
	var wmErr error
	for _, desc := range descs {
		wm, wmErr = api.TextWatermark(text, desc, true, false, types.POINTS)
		if wmErr == nil {
			break
		}
	}
	if wmErr != nil {
		return fmt.Errorf("watermark config: %w", wmErr)
	}
	return api.AddWatermarks(rs, w, nil, wm, conf())
}
