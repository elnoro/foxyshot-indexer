package ocr

import (
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestTesseract_Run(t *testing.T) {
	t.Parallel()
	tt := is.New(t)

	ocr, err := Default()
	tt.NoErr(err)

	t.Run("run on valid image file parses text from the image", func(t *testing.T) {
		res, err := ocr.Run("./testdata/expected-text.jpg")
		tt.NoErr(err)
		tt.Equal(strings.Trim(res, "\n"), "expected text")
	})

	t.Run("run on missing file returns error", func(t *testing.T) {
		res, err := ocr.Run("./testdata/doesnotexist.jpg")
		tt.Equal("", res)   // must return no text in case of error
		tt.True(err != nil) // must return an error
	})
}

func Test_NewTesseractChecksIfCommandIsValid(t *testing.T) {
	t.Parallel()
	tt := is.New(t)

	ocr, err := newTesseract("does-not-exist")

	tt.Equal(nil, ocr)  // must return empty result in case command is unavailable
	tt.True(err != nil) // must return error if command is unavailable
}
