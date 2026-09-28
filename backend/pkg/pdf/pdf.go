package pdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

type Document struct {
	Title        string
	Company      string
	Period       string
	Number       string
	FooterName   string
	Confidential bool
	Lines        []string
}

type Renderer interface {
	Render(doc Document) ([]byte, error)
}

type Fpdf struct{}

func (Fpdf) Render(doc Document) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	company := doc.Company
	if company == "" {
		company = "Aegis"
	}
	footer := doc.FooterName
	if footer == "" {
		footer = "HSE Manager"
	}
	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 8, company, "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		if doc.Number != "" {
			pdf.CellFormat(0, 5, "No: "+doc.Number, "", 1, "L", false, 0, "")
		}
		if doc.Period != "" {
			pdf.CellFormat(0, 5, "Period: "+doc.Period, "", 1, "L", false, 0, "")
		}
		pdf.Ln(2)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetY(-18)
		pdf.SetFont("Arial", "I", 8)
		pdf.CellFormat(0, 5, footer+" - digital sign-off", "", 1, "L", false, 0, "")
		pdf.CellFormat(0, 5, fmt.Sprintf("Page %d", pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	pdf.AddPage()
	if doc.Confidential {
		pdf.SetFont("Arial", "B", 16)
		pdf.SetTextColor(180, 180, 180)
		pdf.CellFormat(0, 10, "CONFIDENTIAL", "", 1, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(2)
	}
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, doc.Title, "", 1, "L", false, 0, "")
	pdf.Ln(2)
	pdf.SetFont("Arial", "", 10)
	for _, line := range doc.Lines {
		pdf.MultiCell(0, 6, strings.TrimSpace(line), "", "L", false)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type Static struct {
	Bytes []byte
	Err   error
}

func (s Static) Render(Document) ([]byte, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	if s.Bytes == nil {
		return []byte("%PDF-stub"), nil
	}
	return s.Bytes, nil
}
