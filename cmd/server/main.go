package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/handlers"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/providers"
	"github.com/cesargomez89/navidrums/internal/repository"
	"github.com/cesargomez89/navidrums/internal/services"
	"github.com/cesargomez89/navidrums/internal/worker"
	"github.com/cesargomez89/navidrums/web"
)

func main() {
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Initialize Logger
	appLogger := logger.New(logger.Config{
		Level:  cfg.LogLevel,
		Format: cfg.LogFormat,
	})

	// Initialize DB
	db, err := repository.NewSQLiteDB(cfg.DBPath)
	if err != nil {
		appLogger.Error("Failed to init DB", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize Provider Manager
	providerManager := providers.NewProviderManager(cfg.ProviderURL)

	// Load saved provider from settings if exists
	settingsRepo := repository.NewSettingsRepo(db)
	if savedProvider, err := settingsRepo.Get(repository.SettingActiveProvider); err == nil && savedProvider != "" {
		providerManager.SetProvider(savedProvider)
	}

	// Initialize Worker
	w := worker.NewWorker(db, providerManager, cfg, appLogger)
	w.Start()
	defer w.Stop()

	// Initialize Services
	jobService := services.NewJobService(db)

	// Initialize Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

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
		w.Write(data)
	}))

	// Routes
	h := handlers.NewHandler(jobService, providerManager, settingsRepo)
	h.RegisterRoutes(r)

	// Start Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
