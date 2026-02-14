package httpapp

import (
	"encoding/json"
	"net/http"

	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/store"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) SearchPage(w http.ResponseWriter, r *http.Request) {
	// Root page
	h.RenderPage(w, "index.html", map[string]interface{}{
		"ActivePage": "search",
	})
}

func (h *Handler) SearchHTMX(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	searchType := r.URL.Query().Get("type")
	if searchType == "" {
		searchType = "album"
	}
	if query == "" {
		w.Write([]byte(""))
		return
	}

	results, err := h.Provider.Search(r.Context(), query, searchType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.RenderFragment(w, "search_results.html", results)
}

func (h *Handler) ArtistPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	artist, err := h.Provider.GetArtist(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Also get Top Tracks if possible?
	// or separate call.

	data := map[string]interface{}{
		"ActivePage": "search",
		"Artist":     artist,
	}
	h.RenderPage(w, "artist.html", data)
}

func (h *Handler) AlbumPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	album, err := h.Provider.GetAlbum(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := map[string]interface{}{
		"ActivePage": "search",
		"Album":      album,
	}
	h.RenderPage(w, "album.html", data)
}

func (h *Handler) PlaylistPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pl, err := h.Provider.GetPlaylist(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := map[string]interface{}{
		"ActivePage": "search",
		"Playlist":   pl,
	}
	h.RenderPage(w, "playlist.html", data)
}

func (h *Handler) DownloadHTMX(w http.ResponseWriter, r *http.Request) {
	jobType := chi.URLParam(r, "type")
	id := chi.URLParam(r, "id")

	_, err := h.JobService.EnqueueJob(id, domain.JobType(jobType))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Return updated queue or confirmation
	w.Write([]byte("<div class='alert alert-success'>Download started!</div>"))
}

func (h *Handler) QueuePage(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.JobService.ListActiveJobs()
	if err != nil {
		h.Logger.Error("Failed to list active jobs", "error", err)
	}
	h.RenderPage(w, "queue.html", map[string]interface{}{
		"ActivePage": "queue",
		"Jobs":       jobs,
	})
}

func (h *Handler) QueueHTMX(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.JobService.ListActiveJobs()
	if err != nil {
		h.Logger.Error("Failed to list active jobs", "error", err)
	}
	h.RenderQueueList(w, jobs)
}

func (h *Handler) HistoryPage(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.JobService.ListFinishedJobs(20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stats, err := h.JobService.GetJobStats()
	if err != nil {
		h.Logger.Error("Failed to get job stats", "error", err)
	}

	h.RenderPage(w, "history.html", map[string]interface{}{
		"ActivePage": "history",
		"Jobs":       jobs,
		"Stats":      stats,
	})
}

func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	h.RenderPage(w, "settings.html", map[string]interface{}{
		"ActivePage": "settings",
	})
}

func (h *Handler) CancelJobHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.JobService.CancelJob(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jobs, err := h.JobService.ListActiveJobs()
	if err != nil {
		h.Logger.Error("Failed to list active jobs", "error", err)
	}
	h.RenderFragment(w, "queue_list.html", jobs)
}

func (h *Handler) RetryJobHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.JobService.RetryJob(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jobs, err := h.JobService.ListActiveJobs()
	if err != nil {
		h.Logger.Error("Failed to list active jobs", "error", err)
	}
	h.RenderFragment(w, "queue_list.html", jobs)
}

func (h *Handler) GetProvidersHTMX(w http.ResponseWriter, r *http.Request) {
	type ProviderData struct {
		Predefined []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"predefined"`
		Custom  []catalog.CustomProvider `json:"custom"`
		Active  string                   `json:"active"`
		Default string                   `json:"default"`
	}

	data := ProviderData{
		Active:  h.ProviderManager.GetBaseURL(),
		Default: h.ProviderManager.GetDefaultURL(),
	}

	customProvidersJSON, err := h.SettingsRepo.Get(store.SettingCustomProviders)
	if err == nil && customProvidersJSON != "" {
		var customProviders []catalog.CustomProvider
		if err := json.Unmarshal([]byte(customProvidersJSON), &customProviders); err == nil {
			data.Custom = customProviders
		}
	}

	customJSON, _ := json.Marshal(data.Custom)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"predefined":` + catalog.GetPredefinedProvidersJSON() + `,"custom":` + string(customJSON) + `,"active":"` + data.Active + `","default":"` + data.Default + `"}`))
}

func (h *Handler) SetProviderHTMX(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url is required", 400)
		return
	}

	h.ProviderManager.SetProvider(url)
	h.SettingsRepo.Set(store.SettingActiveProvider, url)

	w.Write([]byte(`{"success":true,"url":"` + url + `"}`))
}

func (h *Handler) AddCustomProviderHTMX(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	url := r.URL.Query().Get("url")
	if name == "" || url == "" {
		http.Error(w, "name and url are required", 400)
		return
	}

	customProvidersJSON, _ := h.SettingsRepo.Get(store.SettingCustomProviders)
	var customProviders []catalog.CustomProvider
	if customProvidersJSON != "" {
		json.Unmarshal([]byte(customProvidersJSON), &customProviders)
	}

	customProviders = append(customProviders, catalog.CustomProvider{Name: name, URL: url})

	newJSON, _ := json.Marshal(customProviders)
	h.SettingsRepo.Set(store.SettingCustomProviders, string(newJSON))

	w.Write([]byte(`{"success":true}`))
}

func (h *Handler) RemoveCustomProviderHTMX(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url is required", 400)
		return
	}

	customProvidersJSON, err := h.SettingsRepo.Get(store.SettingCustomProviders)
	if err != nil || customProvidersJSON == "" {
		w.Write([]byte(`{"success":false,"error":"no custom catalog"}`))
		return
	}

	var customProviders []catalog.CustomProvider
	if err := json.Unmarshal([]byte(customProvidersJSON), &customProviders); err != nil {
		w.Write([]byte(`{"success":false,"error":"invalid data"}`))
		return
	}

	var newProviders []catalog.CustomProvider
	for _, p := range customProviders {
		if p.URL != url {
			newProviders = append(newProviders, p)
		}
	}

	newJSON, _ := json.Marshal(newProviders)
	h.SettingsRepo.Set(store.SettingCustomProviders, string(newJSON))

	w.Write([]byte(`{"success":true}`))
}

func (h *Handler) SimilarAlbumsHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	albums, err := h.Provider.GetSimilarAlbums(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.RenderFragment(w, "similar_albums.html", albums)
}

func (h *Handler) ClearHistoryHTMX(w http.ResponseWriter, r *http.Request) {
	if err := h.JobService.ClearFinishedJobs(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.RenderFragment(w, "history.html", nil)
}
