package delete

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"urlshortener/internal/lib/logger/sl"
	"urlshortener/internal/storage"
)

// URLDeleter интерфейс для удаления URL
type URLDeleter interface {
	DeleteURL(alias string) error
}

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func New(log *slog.Logger, urlDeleter URLDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.delete.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("alias is empty")
			render.JSON(w, r, Error("alias is required"))
			return
		}

		err := urlDeleter.DeleteURL(alias)
		if err != nil {
			if errors.Is(err, storage.ErrURLNotFound) {
				log.Info("url not found", slog.String("alias", alias))
				render.JSON(w, r, Error("url not found"))
				return
			}

			log.Error("failed to delete url", sl.Err(err))
			render.JSON(w, r, Error("failed to delete url"))
			return
		}

		log.Info("url deleted", slog.String("alias", alias))

		render.JSON(w, r, Response{
			Status: "deleted",
		})
	}
}

// Error возвращает Response с ошибкой
func Error(msg string) Response {
	return Response{
		Status: "error",
		Error:  msg,
	}
}
