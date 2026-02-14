package httpapp

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/cesargomez89/navidrums/web"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	JobService      *app.JobService
	Provider        catalog.Provider
	ProviderManager *catalog.ProviderManager
	SettingsRepo    *store.SettingsRepo
	Templates       *template.Template
	Logger          *logger.Logger
}

func NewHandler(js *app.JobService, pm *catalog.ProviderManager, sr *store.SettingsRepo) *Handler {
	h := &Handler{
		JobService:      js,
		ProviderManager: pm,
		Provider:        pm,
		SettingsRepo:    sr,
		Logger:          logger.Default(),
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
	r.Get("/htmx/queue", h.QueueHTMX)
	r.Post("/htmx/cancel/{id}", h.CancelJobHTMX)
	r.Post("/htmx/retry/{id}", h.RetryJobHTMX)
	r.Get("/history", h.HistoryPage)
	r.Post("/htmx/history/clear", h.ClearHistoryHTMX)
	r.Get("/settings", h.SettingsPage)

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

func (h *Handler) RenderQueueList(w http.ResponseWriter, data interface{}) {
	tmpl, err := template.ParseFS(web.Files, "templates/queue_list.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "queue_list", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
