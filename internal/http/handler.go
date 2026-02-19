package httpapp

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/cesargomez89/navidrums/web"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	JobService       *app.JobService
	DownloadsService *app.DownloadsService
	ProviderManager  *catalog.ProviderManager
	SettingsRepo     *store.SettingsRepo
	Templates        *template.Template
	Logger           *logger.Logger
}

func NewHandler(js *app.JobService, ds *app.DownloadsService, pm *catalog.ProviderManager, sr *store.SettingsRepo) *Handler {
	h := &Handler{
		JobService:       js,
		DownloadsService: ds,
		ProviderManager:  pm,
		SettingsRepo:     sr,
		Logger:           logger.Default(),
	}
	h.ParseTemplates()
	return h
}

func (h *Handler) ParseTemplates() {
	// Not used globally anymore
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.SearchPage)
	r.Get("/htmx/search", h.SearchHTMX)
	r.Get("/artist/{id}", h.ArtistPage)
	r.Get("/album/{id}", h.AlbumPage)
	r.Get("/htmx/album/{id}/similar", h.SimilarAlbumsHTMX)
	r.Get("/playlist/{id}", h.PlaylistPage)

	r.Post("/htmx/download/{type}/{id}", h.DownloadHTMX)
	r.Get("/queue", h.QueuePage)
	r.Get("/htmx/queue/active", h.QueueActiveHTMX)
	r.Get("/htmx/queue/history", h.QueueHistoryHTMX)
	r.Post("/htmx/cancel/{id}", h.CancelJobHTMX)
	r.Post("/htmx/retry/{id}", h.RetryJobHTMX)
	r.Post("/htmx/history/clear", h.ClearHistoryHTMX)
	r.Get("/settings", h.SettingsPage)

	r.Get("/downloads", h.DownloadsPage)
	r.Get("/htmx/downloads", h.DownloadsHTMX)
	r.Delete("/htmx/download/{id}", h.DeleteDownloadHTMX)

	r.Get("/htmx/providers", h.GetProvidersHTMX)
	r.Post("/htmx/provider/set", h.SetProviderHTMX)
	r.Post("/htmx/provider/add", h.AddCustomProviderHTMX)
	r.Post("/htmx/provider/remove", h.RemoveCustomProviderHTMX)
}

func (h *Handler) RenderPage(w http.ResponseWriter, pageTmpl string, data interface{}) {
	// Use ParseFS to properly handle template names
	tmpl, err := template.ParseFS(web.Files,
		"templates/base.html",
		"templates/"+pageTmpl,
		"templates/search_results.html",
		"templates/components/*.html",
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (h *Handler) RenderFragment(w http.ResponseWriter, fragTmpl string, data interface{}) {
	patterns := []string{"templates/components/*.html", "templates/" + fragTmpl}

	tmpl, err := template.ParseFS(web.Files, patterns...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	name := strings.TrimSuffix(filepath.Base(fragTmpl), ".html")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
