package services

import (
	"io"

	enczip "github.com/alexmullins/zip"
)

// ──────────────────────────────────────────────────────────
//  Password-Protect Files — AES-256 Encrypted ZIP
//  Works for PDFs, images, documents, or any file type.
// ──────────────────────────────────────────────────────────

// EncryptedZipFiles creates a password-protected ZIP archive
// containing the given files, using AES encryption.
func EncryptedZipFiles(files []NamedBuffer, password string, w io.Writer) error {
	zw := enczip.NewWriter(w)
	defer zw.Close()

	for _, f := range files {
		fw, err := zw.Encrypt(f.Name, password)
		if err != nil {
			return err
		}
		if _, err := fw.Write(f.Data); err != nil {
			return err
		}
	}

	return zw.Flush()
}
