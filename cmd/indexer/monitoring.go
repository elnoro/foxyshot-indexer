package main

import (
	"net/http"
)

func (app *webApp) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	healthcheck := map[string]string{
		"status":  "available",
		"version": version,
	}

	app.respondJSON(r, w, http.StatusOK, healthcheck)
}
