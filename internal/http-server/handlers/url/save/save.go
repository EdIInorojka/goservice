package save

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"urlshortener/internal/lib/logger/sl"
	"urlshortener/internal/storage"
)

type Request struct {
	URL   string `json:"url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	Status string `json:"status"`
	Alias  string `json:"alias,omitempty"`
	Error  string `json:"error,omitempty"`
}

// URLSaver интерфейс для сохранения URL
type URLSaver interface {
	SaveURL(urlToSave string, alias string) error
}

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.JSON(w, r, Error("failed to decode request"))
			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if req.URL == "" {
			log.Error("url is required")
			render.JSON(w, r, Error("url is required"))
			return
		}

		// Если alias не предоставлен, генерируем его
		if req.Alias == "" {
			// TODO: generate alias
			req.Alias = "example" // временно, нужно реализовать генерацию
		}

		err = urlSaver.SaveURL(req.URL, req.Alias)
		if err != nil {
			if errors.Is(err, storage.ErrURLExists) {
				log.Info("url already exists", slog.String("alias", req.Alias))
				render.JSON(w, r, Error("url already exists"))
				return
			}

			log.Error("failed to save url", sl.Err(err))
			render.JSON(w, r, Error("failed to save url"))
			return
		}

		log.Info("url saved", slog.String("alias", req.Alias))

		render.JSON(w, r, Response{
			Status: "ok",
			Alias:  req.Alias,
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
