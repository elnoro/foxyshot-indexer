package main

import (
	"context"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"time"
)

type webApp struct {
	config Config
	log    *log.Logger
}

func (app *webApp) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthcheck", app.healthcheckHandler)
	mux.Handle("/debug/vars", expvar.Handler())

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

func (app *webApp) healthcheckHandler(w http.ResponseWriter, _ *http.Request) {
	healthcheck := map[string]string{
		"status":  "available",
		"version": version,
	}

	res, err := json.Marshal(healthcheck)
	if err != nil {
		app.log.Println("err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(res)
	if err != nil {
		app.log.Println(err)
	}
}
