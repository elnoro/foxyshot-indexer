package ocr

import (
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestTesseract_Run(t *testing.T) {
	tt := is.New(t)

	ocr, err := Default()
	tt.NoErr(err)

	res, err := ocr.Run("./testdata/expected-text.jpg")
	tt.NoErr(err)
	tt.Equal(strings.Trim(res, "\n"), "expected text")
}
