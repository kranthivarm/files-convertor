package controllers

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"i-lov-pdf/services"
)

// ──────────────────────────────────────────────────────────
//  List Form Fields
// ──────────────────────────────────────────────────────────

func ListFormFieldsHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	fields, err := services.ListFormFields(handle)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": handle.Name,
		"fields":   fields,
		"count":    len(fields),
	})
}

// ──────────────────────────────────────────────────────────
//  Fill Form
// ──────────────────────────────────────────────────────────

func FillFormHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	// Accept JSON data either as form field or as second file
	jsonData := []byte(c.DefaultPostForm("data", ""))
	if len(jsonData) == 0 {
		// Try reading from a second file upload
		fh, fErr := c.FormFile("json")
		if fErr == nil {
			f, oErr := fh.Open()
			if oErr == nil {
				jsonData, _ = io.ReadAll(f)
				f.Close()
			}
		}
	}

	if len(jsonData) == 0 {
		// Try building from individual form fields
		fieldsRaw := strings.TrimSpace(c.DefaultPostForm("fields", ""))
		if fieldsRaw != "" {
			fieldMap := make(map[string]string)
			for _, pair := range strings.Split(fieldsRaw, ";") {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					fieldMap[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			}
			jsonData, err = services.BuildFormFillJSON(fieldMap)
			if err != nil {
				sendError(c, 400, "Could not build form JSON: "+err.Error())
				return
			}
		}
	}

	if len(jsonData) == 0 {
		sendError(c, 400, "Provide form data as JSON (field 'data' or upload 'json' file)")
		return
	}

	safeStream(c, "application/pdf", "filled.pdf", func(w io.Writer) error {
		return services.FillForm(handle, jsonData, w)
	})
	logOp(c, "fill-form", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Flatten Form
// ──────────────────────────────────────────────────────────

func FlattenFormHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	safeStream(c, "application/pdf", "flattened.pdf", func(w io.Writer) error {
		return services.FlattenForm(handle, w)
	})
	logOp(c, "flatten-form", handle.Name)
}

// ──────────────────────────────────────────────────────────
//  Export Form JSON
// ──────────────────────────────────────────────────────────

func ExportFormHandler(c *gin.Context) {
	handle, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}
	defer handle.Close()

	jsonBytes, err := services.ExportFormJSON(handle)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", `attachment; filename="form_data.json"`)
	c.Data(200, "application/json", jsonBytes)
}
