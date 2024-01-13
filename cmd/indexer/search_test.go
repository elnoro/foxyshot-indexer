package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/matryer/is"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSearchHandler(t *testing.T) {
	tt := is.New(t)

	t.Run("valid search", func(t *testing.T) {
		imageDescriptions := &imageRepoMock{
			FindByDescriptionFunc: func(
				ctx context.Context, searchString string, page int, perPage int,
			) ([]domain.Image, error) {
				return []domain.Image{
					{
						FileID:       "any-id",
						Description:  "any-desc",
						LastModified: time.Time{},
					},
				}, nil
			},
		}

		app := newTestApp(imageDescriptions, nil)

		req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(
			`{ "search": "grafana", "page": 11, "per_page": 22 }`,
		))
		w := httptest.NewRecorder()

		app.searchHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		tt.Equal(imageDescriptions.calls.FindByDescription[0].SearchString, "grafana")
		tt.Equal(imageDescriptions.calls.FindByDescription[0].Page, 11)
		tt.Equal(imageDescriptions.calls.FindByDescription[0].PerPage, 22)

		tt.Equal(resp.StatusCode, http.StatusOK)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		bytes.TrimSpace(body)

		tt.Equal(
			string(body),
			"[{\"FileID\":\"any-id\",\"Description\":\"any-desc\",\"LastModified\":\"0001-01-01T00:00:00Z\"}]",
		)
	})

	t.Run("db error", func(t *testing.T) {
		imageDescriptions := &imageRepoMock{
			FindByDescriptionFunc: func(
				ctx context.Context, searchString string, page int, perPage int,
			) ([]domain.Image, error) {
				return []domain.Image{}, errors.New("expected-err")
			},
		}

		app := newTestApp(imageDescriptions, nil)

		req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(
			`{ "search": "grafana", "page": 11, "per_page": 22 }`,
		))
		w := httptest.NewRecorder()

		app.searchHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		tt.Equal(resp.StatusCode, http.StatusInternalServerError)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		bytes.TrimSpace(body)

		tt.Equal(string(body), `{"error": "Internal Server Error"}`)
	})

	t.Run("invalid request values", func(t *testing.T) {
		app := newTestApp(nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()

		app.searchHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		tt.Equal(resp.StatusCode, http.StatusBadRequest)
	})

	t.Run("invalid json in request", func(t *testing.T) {
		app := newTestApp(nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewBufferString(`invalid json`))
		w := httptest.NewRecorder()

		app.searchHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		tt.Equal(resp.StatusCode, http.StatusBadRequest)
	})
}
