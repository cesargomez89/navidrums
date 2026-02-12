package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/providers"
	"github.com/cesargomez89/navidrums/internal/repository"
	"github.com/cesargomez89/navidrums/internal/services"
	"github.com/cesargomez89/navidrums/web"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	JobService      *services.JobService
	Provider        providers.Provider
	ProviderManager *providers.ProviderManager
	SettingsRepo    *repository.SettingsRepo
	Templates       *template.Template
}

func NewHandler(js *services.JobService, pm *providers.ProviderManager, sr *repository.SettingsRepo) *Handler {
	h := &Handler{
		JobService:      js,
		ProviderManager: pm,
		Provider:        pm,
		SettingsRepo:    sr,
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
	r.Get("/playlist/{id}", h.PlaylistPage)

	r.Post("/htmx/download/{type}/{id}", h.DownloadHTMX)
	r.Get("/queue", h.QueuePage)
	r.Get("/htmx/queue", h.QueueHTMX)
	r.Post("/htmx/cancel/{id}", h.CancelJobHTMX)
	r.Get("/history", h.HistoryPage)

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
		"templates/queue_list.html",
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
	// Use ParseFS to properly handle template names
	patterns := []string{"templates/components/*.html", "templates/" + fragTmpl}

	tmpl, err := template.ParseFS(web.Files, patterns...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Execute the specific fragment template
	if err := tmpl.ExecuteTemplate(w, filepath.Base(fragTmpl), data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
