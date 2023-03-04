package main

import (
	"context"
	"net/http"
)

func (app *webApp) deleteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FileID string `json:"file_id" validate:"required"`
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

	err = app.fileStorage.DeleteFile(ctx, req.FileID)
	if err != nil {
		app.serverError(r, w, err)
		return
	}

	err = app.imageDescriptions.Delete(ctx, req.FileID)
	if err != nil {
		app.serverError(r, w, err)
		return
	}

	app.respondNoContent(r, w)
}
