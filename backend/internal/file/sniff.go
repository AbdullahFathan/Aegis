package file

import (
	"bytes"
	"unicode/utf8"
)

const (
	mimeJPEG = "image/jpeg"
	mimePNG  = "image/png"
	mimeWebP = "image/webp"
	mimePDF  = "application/pdf"
	mimeDOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
)

func sniffMIME(header []byte) string {
	if len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return mimeJPEG
	}
	if bytes.HasPrefix(header, []byte{0x89, 0x50, 0x4E, 0x47}) {
		return mimePNG
	}
	if len(header) >= 12 && string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP" {
		return mimeWebP
	}
	if bytes.HasPrefix(header, []byte("%PDF")) {
		return mimePDF
	}
	if bytes.HasPrefix(header, []byte("PK")) {
		return mimeDOCX
	}
	if utf8.Valid(header) {
		return "text/plain"
	}
	return "application/octet-stream"
}

func expectedMIME(ext string) (string, bool) {
	switch ext {
	case ".jpg", ".jpeg":
		return mimeJPEG, true
	case ".png":
		return mimePNG, true
	case ".webp":
		return mimeWebP, true
	case ".pdf":
		return mimePDF, true
	case ".docx":
		return mimeDOCX, true
	default:
		return "", false
	}
}
