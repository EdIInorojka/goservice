package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"

	"urlshortener/internal/config"
	"urlshortener/internal/http-server/handlers/redirect"
	"urlshortener/internal/http-server/handlers/url/delete"
	"urlshortener/internal/http-server/handlers/url/save"
	mwLogger "urlshortener/internal/http-server/middleware/logger"
	"urlshortener/internal/lib/logger/handlers/slogpretty"
	"urlshortener/internal/lib/logger/sl"
	"urlshortener/internal/storage/postgres"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	if cfg.Address == "" {
		cfg.Address = "0.0.0.0:8082"
	}

	log := setupLogger(cfg.Env)

	log.Info(
		"starting url-shortener",
		slog.String("env", cfg.Env),
		slog.String("version", "123"),
	)
	log.Debug("debug messages are enabled")

	// Определяем интерфейс для хранилища
	type Storage interface {
		SaveURL(urlToSave string, alias string) error
		GetURL(alias string) (string, error)
		DeleteURL(alias string) error
		Close() error
	}

	var storage Storage
	var err error

	switch cfg.Storage.Type {
	case "postgres":
		pgCfg := cfg.Storage.Postgres
		storage, err = postgres.New(
			pgCfg.Host,
			pgCfg.Port,
			pgCfg.User,
			pgCfg.Password,
			pgCfg.DBName,
		)
		if err != nil {
			log.Error("failed to init postgres storage", sl.Err(err))
			os.Exit(1)
		}
		log.Info("using PostgreSQL storage")
	default:
		log.Error("unknown storage type", slog.String("type", cfg.Storage.Type))
		os.Exit(1)
	}

	defer func() {
		if err := storage.Close(); err != nil {
			log.Error("failed to close storage", sl.Err(err))
		}
	}()

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	// Public routes
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("URL Shortener API"))
	})
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Protected API routes
	router.Route("/api/v1", func(r chi.Router) {
		// Временно отключаем Basic Auth для тестирования
		// r.Use(middleware.BasicAuth("url-shortener", map[string]string{
		// 	cfg.HTTPServer.User: cfg.HTTPServer.Password,
		// }))

		r.Post("/urls", save.New(log, storage))
		r.Delete("/urls/{alias}", delete.New(log, storage))
	})

	// Redirect route
	router.Get("/{alias}", redirect.New(log, storage))

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("failed to start server", sl.Err(err))
		}
	}()

	log.Info("server started")

	<-done
	log.Info("stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("failed to stop server", sl.Err(err))
		return
	}

	log.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
