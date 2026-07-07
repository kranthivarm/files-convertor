package services

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// ──────────────────────────────────────────────────────────
//  List Form Fields
// ──────────────────────────────────────────────────────────

func ListFormFields(rs io.ReadSeeker) ([]FormFieldInfo, error) {
	fields, err := api.FormFields(rs, conf())
	if err != nil {
		return nil, err
	}

	var out []FormFieldInfo
	for _, f := range fields {
		out = append(out, FormFieldInfo{
			Name:    f.Name,
			Type:    f.Typ.String(),
			Value:   f.V,
			Default: f.Dv,
			Locked:  f.Locked,
		})
	}
	return out, nil
}

// ──────────────────────────────────────────────────────────
//  Fill Form — accepts JSON data
// ──────────────────────────────────────────────────────────

func FillForm(rs io.ReadSeeker, jsonData []byte, w io.Writer) error {
	rd := bytes.NewReader(jsonData)
	return api.FillForm(rs, rd, w, conf())
}

// ──────────────────────────────────────────────────────────
//  Flatten Form — lock all fields + optimize
// ──────────────────────────────────────────────────────────

func FlattenForm(rs io.ReadSeeker, w io.Writer) error {
	fields, err := api.FormFields(rs, conf())
	if err != nil {
		return api.Optimize(rs, w, conf())
	}

	fieldNames := make([]string, 0, len(fields))
	for _, f := range fields {
		fieldNames = append(fieldNames, f.Name)
	}

	if len(fieldNames) == 0 {
		return api.Optimize(rs, w, conf())
	}

	var buf bytes.Buffer
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := api.LockFormFields(rs, &buf, fieldNames, conf()); err != nil {
		return err
	}

	locked := bytes.NewReader(buf.Bytes())
	return api.Optimize(locked, w, conf())
}

// ──────────────────────────────────────────────────────────
//  Export Form Fields as JSON
// ──────────────────────────────────────────────────────────

func ExportFormJSON(rs io.ReadSeeker) ([]byte, error) {
	var buf bytes.Buffer
	if err := api.ExportFormJSON(rs, &buf, "form", conf()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ──────────────────────────────────────────────────────────
//  Build Form Fill JSON
// ──────────────────────────────────────────────────────────

type FormFillData struct {
	TextField []FormTextField `json:"textfield,omitempty"`
}

type FormTextField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func BuildFormFillJSON(fields map[string]string) ([]byte, error) {
	data := FormFillData{}
	for name, value := range fields {
		data.TextField = append(data.TextField, FormTextField{Name: name, Value: value})
	}
	wrapped := map[string]interface{}{
		"forms": []FormFillData{data},
	}
	return json.Marshal(wrapped)
}
