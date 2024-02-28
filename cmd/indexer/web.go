package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"github.com/elnoro/foxyshot-indexer/internal/indexer"
	"log"
	"net/http"
	"time"

	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/elnoro/foxyshot-indexer/internal/monitoring"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//go:generate moq -out web_moq_test.go . imageRepo fileStorage
type imageRepo interface {
	FindByDescription(ctx context.Context, searchString string, page, perPage int) ([]domain.Image, error)
	FindByEmbedding(ctx context.Context, embedding domain.Embedding, page, perPage int) ([]domain.Image, error)
	Delete(ctx context.Context, fileID string) error
}

type fileStorage interface {
	DeleteFile(ctx context.Context, key string) error
}

type webApp struct {
	config Config
	log    *log.Logger

	imageDescriptions imageRepo
	embeddings        indexer.ImageEmbeddingClient
	fileStorage       fileStorage

	tracker *monitoring.Tracker
}

func (app *webApp) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(httprate.LimitByRealIP(10, 10*time.Second))
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/healthcheck", app.healthcheckHandler)

	r.Method(http.MethodGet, "/debug/vars", expvar.Handler())
	r.Method(http.MethodGet, "/metrics", promhttp.Handler())

	r.Route("/api", func(r chi.Router) {
		r.Post("/search", app.searchHandler)
		r.Post("/image-search", app.imageSearchHandler)
		r.Delete("/delete", app.deleteHandler)
	})

	r.NotFound(app.notFound)
	r.MethodNotAllowed(app.methodNotAllowed)

	return r
}

func (app *webApp) serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.Port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownError := make(chan error)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		shutdownError <- srv.Shutdown(shutdownCtx)
	}()

	app.log.Println("starting server on port", app.config.Port)
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server listen&server err, %w", err)
	}

	err = <-shutdownError
	if err != nil {
		return fmt.Errorf("server shutdown err, %w", err)
	}

	app.log.Println("server stopped")

	return nil
}
