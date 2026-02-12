package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	// Initialize Provider (Mock for now, or real)
	// We can switch based on env or config
	var provider providers.Provider
	if os.Getenv("USE_MOCK") == "true" {
		provider = providers.NewMockProvider()
	} else {
		provider = providers.NewHifiProvider(cfg.ProviderURL)
	}

	// Initialize Worker
	w := worker.NewWorker(db, provider, cfg, appLogger)
	w.Start()
	defer w.Stop()

	// Initialize Services
	jobService := services.NewJobService(db)

	// Initialize Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Serve Static Files
	// Assuming web/static exists
	fileServer := http.FileServer(http.Dir("./web/static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// Routes
	h := handlers.NewHandler(jobService, provider)
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
