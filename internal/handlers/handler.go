package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/cesargomez89/navidrums/internal/providers"
	"github.com/cesargomez89/navidrums/internal/services"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	JobService *services.JobService
	Provider   providers.Provider
	Templates  *template.Template
}

func NewHandler(js *services.JobService, p providers.Provider) *Handler {
	h := &Handler{
		JobService: js,
		Provider:   p,
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
	r.Get("/history", h.HistoryPage)
}

func (h *Handler) RenderPage(w http.ResponseWriter, pageTmpl string, data interface{}) {
	// Include all necessary fragments that might be used
	files := []string{"web/templates/base.html", "web/templates/" + pageTmpl, "web/templates/queue_list.html", "web/templates/search_results.html"}
	compFiles, _ := filepath.Glob("web/templates/components/*.html")
	files = append(files, compFiles...)

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (h *Handler) RenderFragment(w http.ResponseWriter, fragTmpl string, data interface{}) {
	// Parse the fragment and all components to support inclusion
	files, _ := filepath.Glob("web/templates/components/*.html")
	files = append(files, "web/templates/"+fragTmpl)

	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// Execute the specific fragment template
	if err := tmpl.ExecuteTemplate(w, filepath.Base(fragTmpl), data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
