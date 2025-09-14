package delete

import (
	"errors"
	"net/http"

	"urlshortener/internal/lib/api/response"
	"urlshortener/internal/lib/logger/sl"
	"urlshortener/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
)

type Request struct {
	Alias string `json:"alias"`
}

type Response struct {
	response.Response
}

//go:generate go run github.com/vektra/mockery/v2@v2.28.2 --name=URLDeleter
type URLDeleter interface {
	DeleteURL(alias string) error
}

func New(log *slog.Logger, urlDeleter URLDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.delete.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.JSON(w, r, response.Error("failed to decode request"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if req.Alias == "" {
			log.Info("alias is empty")

			render.JSON(w, r, response.Error("alias is required"))

			return
		}

		err = urlDeleter.DeleteURL(req.Alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", slog.String("alias", req.Alias))

			render.JSON(w, r, response.Error("url not found"))

			return
		}
		if err != nil {
			log.Error("failed to delete url", sl.Err(err))

			render.JSON(w, r, response.Error("failed to delete url"))

			return
		}

		log.Info("url deleted", slog.String("alias", req.Alias))

		render.JSON(w, r, Response{
			Response: response.OK(),
		})
	}
}
