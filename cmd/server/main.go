package main

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/downloader"
	httpapp "github.com/cesargomez89/navidrums/internal/http"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/cesargomez89/navidrums/web"
)

func main() {
	cfg := config.Load()

	// Initialize Logger
	appLogger := logger.New(logger.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		appLogger.Error("Configuration error", "error", err)
		os.Exit(1)
	}

	// Initialize DB
	db, err := store.NewSQLiteDB(cfg.DBPath)
	if err != nil {
		appLogger.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			appLogger.Error("Failed to close DB", "error", closeErr)
		}
	}()

	// Initialize Provider Manager
	providerManager := catalog.NewProviderManager(cfg.ProviderURL, db, cfg.CacheTTL, appLogger)

	// Load saved provider from settings if exists
	settingsRepo := store.NewSettingsRepo(db)
	if savedProvider, err := settingsRepo.Get(store.SettingActiveProvider); err == nil && savedProvider != "" {
		providerManager.SetProvider(savedProvider)
	}

	// Initialize Worker
	w := downloader.NewWorker(db, providerManager, cfg, appLogger)
	w.Start()
	defer w.Stop()

	// Initialize Services
	jobService := app.NewJobService(db, appLogger)
	downloadsService := app.NewDownloadsService(db, appLogger)

	// Initialize Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Basic Auth Middleware
	if cfg.Password != "" {
		r.Use(basicAuthMiddleware(cfg.Username, cfg.Password))
	}

	// Serve Static Files from embedded filesystem
	r.Handle("/static/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := "static" + r.URL.Path[len("/static"):]
		data, err := web.Files.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		contentType := "application/octet-stream"
		switch {
		case strings.HasSuffix(path, ".css"):
			contentType = "text/css"
		case strings.HasSuffix(path, ".js"):
			contentType = "application/javascript"
		case strings.HasSuffix(path, ".png"):
			contentType = "image/png"
		case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
			contentType = "image/jpeg"
		case strings.HasSuffix(path, ".svg"):
			contentType = "image/svg+xml"
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
	}))

	// Routes
	h := httpapp.NewHandler(jobService, downloadsService, providerManager, settingsRepo)
	h.RegisterRoutes(r)

	// Start Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		appLogger.Info("Server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server error", "error", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", "error", err)
	}

	appLogger.Info("Server exiting")
}

func basicAuthMiddleware(username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="Navidrums"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
