package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

func (app *webApp) respondJSON(r *http.Request, w http.ResponseWriter, status int, resp any) {
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		app.serverError(r, w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(jsonResp)
	if err != nil {
		app.log.Println("Error:", err, "URL:", r.URL.String())
	}
}

func (app *webApp) respondNoContent(_ *http.Request, w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func (app *webApp) parseJSON(r *http.Request, o any) error {
	err := json.NewDecoder(r.Body).Decode(o)
	if err != nil {
		return fmt.Errorf("decoding request, %w", err)
	}
	return nil
}

func (app *webApp) malformedJSON(r *http.Request, w http.ResponseWriter) {
	app.errorResponse(r, w, http.StatusBadRequest, "malformed json")
}

func (app *webApp) validationError(r *http.Request, w http.ResponseWriter, err error) {
	app.errorResponse(r, w, http.StatusBadRequest, err.Error())
}

func (app *webApp) serverError(r *http.Request, w http.ResponseWriter, err error) {
	app.log.Println("Error:", err, "URL:", r.URL.String())

	app.errorResponse(r, w, http.StatusInternalServerError, "Internal Server Error")
}

func (app *webApp) errorResponse(r *http.Request, w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_, err := fmt.Fprintf(w, `{"error": "%s"}`, message)
	if err != nil {
		app.error(r, err)
	}
}

func (app *webApp) error(r *http.Request, err error) {
	app.log.Println("Error:", err, "URL:", r.URL.String())
}

func (app *webApp) validate(o any) error {
	validate := validator.New()

	return validate.Struct(o)
}

func (app *webApp) notFound(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(r, w, http.StatusNotFound, "404 Not Found")
}

func (app *webApp) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	app.errorResponse(r, w, http.StatusMethodNotAllowed, "Method Not Allowed")
}
