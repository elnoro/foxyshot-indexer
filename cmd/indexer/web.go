package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/elnoro/foxyshot-indexer/internal/domain"
	"github.com/elnoro/foxyshot-indexer/internal/monitoring"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type imageRepo interface {
	Search(ctx context.Context, searchString string, page, perPage int) ([]domain.Image, error)
	Delete(ctx context.Context, fileID string) error
}

type fileStorage interface {
	DeleteFile(ctx context.Context, key string) error
}

type webApp struct {
	config Config
	log    *log.Logger

	imageDescriptions imageRepo
	fileStorage       fileStorage

	tracker *monitoring.Tracker
}

func (app *webApp) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthcheck", app.healthcheckHandler)
	mux.Handle("/debug/vars", expvar.Handler())
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/api/search", app.searchHandler)
	mux.HandleFunc("/api/delete", app.deleteHandler)

	return mux
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
