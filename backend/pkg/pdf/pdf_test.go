package pdf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFpdfRenderContainsTitle(t *testing.T) {
	b, err := Fpdf{}.Render(Document{
		Title:        "Rekap Insiden Bulanan",
		Company:      "PT Demo",
		Period:       "2026-08",
		FooterName:   "HSE Manager",
		Confidential: true,
		Lines:        []string{"INC-2026-08-0001 | Slip"},
	})
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(b, []byte("%PDF")))
	require.Greater(t, len(b), 200)
}
