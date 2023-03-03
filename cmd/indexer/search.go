package main

import (
	"context"
	"net/http"
)

func (app *webApp) searchHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Search  string `json:"search" validate:"required"`
		Page    int    `json:"page" validate:"min=1"`
		PerPage int    `json:"per_page" validate:"min=1,max=100"`
	}
	err := app.parseJSON(r, &req)
	if err != nil {
		app.malformedJSON(r, w)
		return
	}

	err = app.validate(req)
	if err != nil {
		app.validationError(r, w, err)
		return
	}

	ctx := context.Background()
	images, err := app.imageSearcher.Search(ctx, req.Search, req.Page, req.PerPage)
	if err != nil {
		app.serverError(r, w, err)
		return
	}

	app.respondJSON(r, w, http.StatusOK, images)
}
