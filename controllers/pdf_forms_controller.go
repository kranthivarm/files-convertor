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
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	fields, err := services.ListFormFields(rs)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": origName,
		"fields":   fields,
		"count":    len(fields),
	})
}

// ──────────────────────────────────────────────────────────
//  Fill Form
// ──────────────────────────────────────────────────────────

func FillFormHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

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

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="filled.pdf"`)
	c.Status(200)
	if err := services.FillForm(rs, jsonData, c.Writer); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "fill-form", origName)
}

// ──────────────────────────────────────────────────────────
//  Flatten Form
// ──────────────────────────────────────────────────────────

func FlattenFormHandler(c *gin.Context) {
	rs, origName, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="flattened.pdf"`)
	c.Status(200)
	if err := services.FlattenForm(rs, c.Writer); err != nil {
		c.Error(err)
		return
	}
	logOp(c, "flatten-form", origName)
}

// ──────────────────────────────────────────────────────────
//  Export Form JSON
// ──────────────────────────────────────────────────────────

func ExportFormHandler(c *gin.Context) {
	rs, _, err := readOne(c, "file")
	if err != nil {
		sendError(c, 400, err.Error())
		return
	}

	jsonBytes, err := services.ExportFormJSON(rs)
	if err != nil {
		sendError(c, 500, err.Error())
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", `attachment; filename="form_data.json"`)
	c.Data(200, "application/json", jsonBytes)
}
