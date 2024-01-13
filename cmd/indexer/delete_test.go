package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/matryer/is"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteHandler(t *testing.T) {
	tt := is.New(t)

	t.Run("successful deletion", func(t *testing.T) {
		imageDescriptions := &imageRepoMock{
			DeleteFunc: func(ctx context.Context, fileID string) error { return nil },
		}

		storage := &fileStorageMock{
			DeleteFileFunc: func(ctx context.Context, key string) error { return nil },
		}

		app := newTestApp(imageDescriptions, storage)

		req := httptest.NewRequest(http.MethodPost, "/delete", bytes.NewBufferString(
			`{ "file_id": "expected-file-id" }`,
		))
		w := httptest.NewRecorder()

		app.deleteHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		tt.Equal(imageDescriptions.calls.Delete[0].FileID, "expected-file-id")
		tt.Equal(storage.calls.DeleteFile[0].Key, "expected-file-id")

		tt.Equal(resp.StatusCode, http.StatusNoContent)
	})

	t.Run("storage error", func(t *testing.T) {
		imageDescriptions := &imageRepoMock{DeleteFunc: func(ctx context.Context, fileID string) error {
			return nil
		}}
		fs := &fileStorageMock{
			DeleteFileFunc: func(ctx context.Context, key string) error { return errors.New("delete err") },
		}

		app := newTestApp(imageDescriptions, fs)

		req := httptest.NewRequest(http.MethodPost, "/delete", bytes.NewBufferString(
			`{ "file_id": "expected-file-id"}`,
		))
		w := httptest.NewRecorder()

		app.deleteHandler(w, req)

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

	t.Run("db error", func(t *testing.T) {
		imageDescriptions := &imageRepoMock{DeleteFunc: func(ctx context.Context, fileID string) error {
			return errors.New("delete err")
		}}
		fs := &fileStorageMock{
			DeleteFileFunc: func(ctx context.Context, key string) error { return nil },
		}

		app := newTestApp(imageDescriptions, fs)

		req := httptest.NewRequest(http.MethodPost, "/delete", bytes.NewBufferString(
			`{ "file_id": "expected-file-id"}`,
		))
		w := httptest.NewRecorder()

		app.deleteHandler(w, req)

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

		req := httptest.NewRequest(http.MethodPost, "/delete", bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()

		app.deleteHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		tt.Equal(resp.StatusCode, http.StatusBadRequest)
	})

	t.Run("invalid json in request", func(t *testing.T) {
		app := newTestApp(nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/delete", bytes.NewBufferString(`invalid json`))
		w := httptest.NewRecorder()

		app.deleteHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		tt.Equal(resp.StatusCode, http.StatusBadRequest)
	})
}
