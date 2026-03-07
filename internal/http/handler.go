package httpapp

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/form/v4"

	"github.com/cesargomez89/navidrums/internal/app"
	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/config"
	"github.com/cesargomez89/navidrums/internal/logger"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/cesargomez89/navidrums/web"
)

type Handler struct {
	cachedRecsTime   time.Time
	JobService       *app.JobService
	DownloadsService *app.DownloadsService
	ProviderManager  *catalog.ProviderManager
	SettingsRepo     *store.SettingsRepo
	Config           *config.Config
	Templates        *template.Template
	Logger           *logger.Logger
	FormDecoder      *form.Decoder
	cachedRecs       *RecommendationsData
	Theme            string
	recsMutex        sync.RWMutex
}

func NewHandler(js *app.JobService, ds *app.DownloadsService, pm *catalog.ProviderManager, sr *store.SettingsRepo, cfg *config.Config) *Handler {
	h := &Handler{
		JobService:       js,
		DownloadsService: ds,
		ProviderManager:  pm,
		SettingsRepo:     sr,
		Config:           cfg,
		Logger:           logger.Default(),
		FormDecoder:      form.NewDecoder(),
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
	r.Get("/htmx/lucky", h.LuckyHTMX)
	r.Get("/artist/{id}", h.ArtistPage)
	r.Get("/album/{id}", h.AlbumPage)
	r.Get("/htmx/album/{id}/similar", h.SimilarAlbumsHTMX)
	r.Get("/htmx/artist/{id}/similar", h.SimilarArtistsHTMX)
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
	r.Post("/htmx/downloads/sync", h.SyncAllHTMX)
	r.Post("/htmx/downloads/bulk-delete", h.BulkDeleteHTMX)
	r.Post("/htmx/downloads/bulk-sync", h.BulkSyncHTMX)
	r.Post("/htmx/downloads/bulk-genre", h.BulkUpdateGenreHTMX)
	r.Delete("/htmx/download/{id}", h.DeleteDownloadHTMX)

	r.Get("/stream/{id}", h.StreamTrack)

	r.Get("/track/{id}", h.TrackPage)
	r.Get("/htmx/track/{id}", h.TrackHTMX)
	r.Post("/htmx/track/{id}/save", h.SaveTrackHTMX)
	r.Post("/htmx/track/{id}/sync", h.SyncTrackHTMX)
	r.Post("/htmx/track/{id}/enrich", h.EnrichTrackHTMX)
	r.Post("/htmx/track/{id}/enrich-hifi", h.EnrichHiFiHTMX)

	r.Get("/htmx/providers", h.GetProvidersHTMX)
	r.Post("/htmx/provider/set", h.SetProviderHTMX)
	r.Post("/htmx/provider/add", h.AddCustomProviderHTMX)
	r.Post("/htmx/provider/remove", h.RemoveCustomProviderHTMX)

	r.Get("/htmx/genre-map", h.GetGenreMapHTMX)
	r.Post("/htmx/genre-map", h.SetGenreMapHTMX)
	r.Post("/htmx/genre-map/reset", h.ResetGenreMapHTMX)

	r.Get("/htmx/genre-separator", h.GetGenreSeparatorHTMX)
	r.Post("/htmx/genre-separator", h.SetGenreSeparatorHTMX)

	r.Get("/htmx/theme", h.GetThemeHTMX)
	r.Post("/htmx/theme", h.SetThemeHTMX)
	r.Post("/htmx/theme/reset", h.ResetThemeHTMX)
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

	// Inject global theme if not already set in data
	if m, ok := data.(map[string]interface{}); ok {
		if _, exists := m["Theme"]; !exists {
			theme, err := h.SettingsRepo.Get(store.SettingTheme)
			if err != nil || theme == "" {
				theme = h.Theme
			}
			m["Theme"] = theme
		}
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
