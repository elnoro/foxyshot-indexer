package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matryer/is"
)

func Test_webApp_searchHandler(t *testing.T) {
	app := &webApp{}

	tt := is.New(t)

	t.Run("empty request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/search", nil)
		w := httptest.NewRecorder()
		app.searchHandler(w, req)

		tt.Equal(w.Code, http.StatusBadRequest)
	})

	t.Run("invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/search",
			io.NopCloser(strings.NewReader(`{"invalid" "json"}`)),
		)

		w := httptest.NewRecorder()
		app.searchHandler(w, req)

		tt.Equal(w.Code, http.StatusBadRequest)
	})

	t.Run("validation fails", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/search",
			io.NopCloser(strings.NewReader(`{"search": "", "page": 0, "per_page": 0}`)),
		)

		w := httptest.NewRecorder()
		app.searchHandler(w, req)

		tt.Equal(w.Code, http.StatusBadRequest)
		tt.Equal(w.Body.String(), `{"error": "Key: 'Search' Error:Field validation for 'Search' failed on the 'required' tag
Key: 'Page' Error:Field validation for 'Page' failed on the 'min' tag
Key: 'PerPage' Error:Field validation for 'PerPage' failed on the 'min' tag"}`)
	})
}
