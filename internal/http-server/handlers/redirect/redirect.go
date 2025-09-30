package redirect

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"urlshortener/internal/lib/logger/sl"
	"urlshortener/internal/storage"
)

// URLGetter интерфейс для получения URL
type URLGetter interface {
	GetURL(alias string) (string, error)
}

func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.redirect.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("alias is empty")
			http.Error(w, "alias is required", http.StatusBadRequest)
			return
		}

		url, err := urlGetter.GetURL(alias)
		if err != nil {
			if errors.Is(err, storage.ErrURLNotFound) {
				log.Info("url not found", slog.String("alias", alias))
				http.Error(w, "url not found", http.StatusNotFound)
				return
			}

			log.Error("failed to get url", sl.Err(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		log.Info("redirecting", slog.String("alias", alias), slog.String("url", url))

		http.Redirect(w, r, url, http.StatusFound)
	}
}
