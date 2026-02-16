package httpapp

import (
	"encoding/json"
	"net/http"

	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/constants"
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated queue or confirmation
	w.Write([]byte("<div class='alert alert-success'>Download started!</div>"))
}

func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	h.RenderPage(w, "settings.html", map[string]interface{}{
		"ActivePage": "settings",
	})
}

func (h *Handler) QueuePage(w http.ResponseWriter, r *http.Request) {
	h.RenderPage(w, "queue.html", map[string]interface{}{
		"ActivePage": "queue",
	})
}

func (h *Handler) QueueActiveHTMX(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.JobService.ListActiveJobs()
	if err != nil {
		h.Logger.Error("Failed to list active jobs", "error", err)
	}
	h.RenderFragment(w, "components/active_tab.html", map[string]interface{}{
		"ActiveJobs": jobs,
	})
}

func (h *Handler) QueueHistoryHTMX(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.JobService.ListFinishedJobs(constants.MaxHistoryItems)
	if err != nil {
		h.Logger.Error("Failed to list finished jobs", "error", err)
		return
	}

	stats, err := h.JobService.GetJobStats()
	if err != nil {
		h.Logger.Error("Failed to get job stats", "error", err)
	}

	h.RenderFragment(w, "components/history_tab.html", map[string]interface{}{
		"HistoryJobs": jobs,
		"Stats":       stats,
	})
}

func (h *Handler) CancelJobHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.JobService.CancelJob(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jobs, err := h.JobService.ListActiveJobs()
	if err != nil {
		h.Logger.Error("Failed to list active jobs", "error", err)
	}
	h.RenderFragment(w, "components/active_tab.html", map[string]interface{}{
		"ActiveJobs": jobs,
	})
}

func (h *Handler) RetryJobHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.JobService.RetryJob(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jobs, err := h.JobService.ListActiveJobs()
	if err != nil {
		h.Logger.Error("Failed to list active jobs", "error", err)
	}
	h.RenderFragment(w, "components/active_tab.html", map[string]interface{}{
		"ActiveJobs": jobs,
	})
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
		if err := json.Unmarshal([]byte(customProvidersJSON), &customProviders); err != nil {
			h.Logger.Error("Failed to unmarshal custom providers", "error", err)
		} else {
			data.Custom = customProviders
		}
	}

	response := map[string]interface{}{
		"predefined": json.RawMessage(catalog.GetPredefinedProvidersJSON()),
		"custom":     data.Custom,
		"active":     data.Active,
		"default":    data.Default,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Logger.Error("Failed to encode providers response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) SetProviderHTMX(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	h.ProviderManager.SetProvider(url)
	if err := h.SettingsRepo.Set(store.SettingActiveProvider, url); err != nil {
		h.Logger.Error("Failed to save active provider", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(`{"success":true,"url":"` + url + `"}`))
}

func (h *Handler) AddCustomProviderHTMX(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	url := r.URL.Query().Get("url")
	if name == "" || url == "" {
		http.Error(w, "name and url are required", http.StatusBadRequest)
		return
	}

	customProvidersJSON, err := h.SettingsRepo.Get(store.SettingCustomProviders)
	if err != nil {
		h.Logger.Error("Failed to get custom providers", "error", err)
	}
	var customProviders []catalog.CustomProvider
	if customProvidersJSON != "" {
		if err := json.Unmarshal([]byte(customProvidersJSON), &customProviders); err != nil {
			h.Logger.Error("Failed to unmarshal custom providers", "error", err)
		}
	}

	customProviders = append(customProviders, catalog.CustomProvider{Name: name, URL: url})

	newJSON, err := json.Marshal(customProviders)
	if err != nil {
		h.Logger.Error("Failed to marshal custom providers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := h.SettingsRepo.Set(store.SettingCustomProviders, string(newJSON)); err != nil {
		h.Logger.Error("Failed to save custom providers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(`{"success":true}`))
}

func (h *Handler) RemoveCustomProviderHTMX(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
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

	newJSON, err := json.Marshal(newProviders)
	if err != nil {
		h.Logger.Error("Failed to marshal custom providers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := h.SettingsRepo.Set(store.SettingCustomProviders, string(newJSON)); err != nil {
		h.Logger.Error("Failed to save custom providers", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

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

	h.QueueHistoryHTMX(w, r)
}

func (h *Handler) DownloadsPage(w http.ResponseWriter, r *http.Request) {
	h.RenderPage(w, "downloads.html", map[string]interface{}{
		"ActivePage": "downloads",
	})
}

func (h *Handler) DownloadsHTMX(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	var tracks []*domain.Track
	var err error

	if query != "" {
		tracks, err = h.DownloadsService.SearchDownloads(query)
	} else {
		tracks, err = h.DownloadsService.ListDownloads()
	}
	if err != nil {
		h.Logger.Error("Failed to list downloads", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.RenderFragment(w, "components/downloads_list.html", map[string]interface{}{
		"Downloads": tracks,
	})
}

func (h *Handler) DeleteDownloadHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.DownloadsService.DeleteDownload(id); err != nil {
		h.Logger.Error("Failed to delete download", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.DownloadsHTMX(w, r)
}
