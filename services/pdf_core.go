package services

import (
	"archive/zip"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ──────────────────────────────────────────────────────────
//  Shared configuration
// ──────────────────────────────────────────────────────────

func conf() *model.Configuration {
	c := model.NewDefaultConfiguration()
	c.ValidationMode = model.ValidationRelaxed
	return c
}

// ──────────────────────────────────────────────────────────
//  Shared types
// ──────────────────────────────────────────────────────────

// NamedBuffer holds a filename and byte content for zipping.
type NamedBuffer struct {
	Name string
	Data []byte
}

// ImageData holds an extracted image's bytes and filename.
type ImageData struct {
	Name string
	Data []byte
}

// CompareResult holds the result of comparing two PDFs.
type CompareResult struct {
	File1Pages  int              `json:"file1_pages"`
	File2Pages  int              `json:"file2_pages"`
	PagesMatch  bool             `json:"pages_match"`
	File1Title  string           `json:"file1_title"`
	File2Title  string           `json:"file2_title"`
	File1Author string           `json:"file1_author"`
	File2Author string           `json:"file2_author"`
	PageDiffs   []PageDiffEntry  `json:"page_diffs,omitempty"`
	Summary     string           `json:"summary"`
}

type PageDiffEntry struct {
	Page       int    `json:"page"`
	Difference string `json:"difference"`
}

// FormFieldInfo describes a single form field.
type FormFieldInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Value   string `json:"value,omitempty"`
	Default string `json:"default,omitempty"`
	Locked  bool   `json:"locked"`
}

// ──────────────────────────────────────────────────────────
//  ZIP helpers
// ──────────────────────────────────────────────────────────

func ZipNamedBuffers(items []NamedBuffer, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()
	for _, item := range items {
		fw, err := zw.Create(item.Name)
		if err != nil {
			return err
		}
		if _, err := fw.Write(item.Data); err != nil {
			return err
		}
	}
	return nil
}

func ZipImageData(items []ImageData, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()
	for _, item := range items {
		fw, err := zw.Create(item.Name)
		if err != nil {
			return err
		}
		if _, err := fw.Write(item.Data); err != nil {
			return err
		}
	}
	return nil
}
