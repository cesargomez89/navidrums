package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cesargomez89/navidrums/internal/models"
	"github.com/cesargomez89/navidrums/internal/providers"
	"github.com/cesargomez89/navidrums/internal/repository"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) SearchPage(w http.ResponseWriter, r *http.Request) {
	// Root page
	h.RenderPage(w, "index.html", nil)
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

	h.RenderPage(w, "artist.html", artist)
}

func (h *Handler) AlbumPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	album, err := h.Provider.GetAlbum(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	h.RenderPage(w, "album.html", album)
}

func (h *Handler) PlaylistPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pl, err := h.Provider.GetPlaylist(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	h.RenderPage(w, "playlist.html", pl)
}

func (h *Handler) DownloadHTMX(w http.ResponseWriter, r *http.Request) {
	jobType := chi.URLParam(r, "type")
	id := chi.URLParam(r, "id")

	_, err := h.JobService.EnqueueJob(id, models.JobType(jobType))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Return updated queue or confirmation
	w.Write([]byte("<div class='alert alert-success'>Download started!</div>"))
}

func (h *Handler) QueuePage(w http.ResponseWriter, r *http.Request) {
	jobs, _ := h.JobService.Repo.ListActiveJobs() // Use db directly for simplicity
	h.RenderPage(w, "queue.html", jobs)
}

func (h *Handler) QueueHTMX(w http.ResponseWriter, r *http.Request) {
	jobs, _ := h.JobService.Repo.ListActiveJobs()
	h.RenderQueueList(w, jobs)
}

func (h *Handler) HistoryPage(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.JobService.Repo.ListFinishedJobs(20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.RenderPage(w, "history.html", jobs)
}

func (h *Handler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	h.RenderPage(w, "settings.html", nil)
}

func (h *Handler) CancelJobHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.JobService.CancelJob(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Return updated queue
	jobs, _ := h.JobService.ListActiveJobs()
	h.RenderFragment(w, "queue_list.html", jobs)
}

func (h *Handler) GetProvidersHTMX(w http.ResponseWriter, r *http.Request) {
	type ProviderData struct {
		Predefined []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"predefined"`
		Custom  []providers.CustomProvider `json:"custom"`
		Active  string                     `json:"active"`
		Default string                     `json:"default"`
	}

	data := ProviderData{
		Active:  h.ProviderManager.GetBaseURL(),
		Default: h.ProviderManager.GetDefaultURL(),
	}

	customProvidersJSON, err := h.SettingsRepo.Get(repository.SettingCustomProviders)
	if err == nil && customProvidersJSON != "" {
		var customProviders []providers.CustomProvider
		if err := json.Unmarshal([]byte(customProvidersJSON), &customProviders); err == nil {
			data.Custom = customProviders
		}
	}

	customJSON, _ := json.Marshal(data.Custom)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"predefined":` + providers.GetPredefinedProvidersJSON() + `,"custom":` + string(customJSON) + `,"active":"` + data.Active + `","default":"` + data.Default + `"}`))
}

func (h *Handler) SetProviderHTMX(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url is required", 400)
		return
	}

	h.ProviderManager.SetProvider(url)
	h.SettingsRepo.Set(repository.SettingActiveProvider, url)

	w.Write([]byte(`{"success":true,"url":"` + url + `"}`))
}

func (h *Handler) AddCustomProviderHTMX(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	url := r.URL.Query().Get("url")
	if name == "" || url == "" {
		http.Error(w, "name and url are required", 400)
		return
	}

	customProvidersJSON, _ := h.SettingsRepo.Get(repository.SettingCustomProviders)
	var customProviders []providers.CustomProvider
	if customProvidersJSON != "" {
		json.Unmarshal([]byte(customProvidersJSON), &customProviders)
	}

	customProviders = append(customProviders, providers.CustomProvider{Name: name, URL: url})

	newJSON, _ := json.Marshal(customProviders)
	h.SettingsRepo.Set(repository.SettingCustomProviders, string(newJSON))

	w.Write([]byte(`{"success":true}`))
}

func (h *Handler) RemoveCustomProviderHTMX(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url is required", 400)
		return
	}

	customProvidersJSON, err := h.SettingsRepo.Get(repository.SettingCustomProviders)
	if err != nil || customProvidersJSON == "" {
		w.Write([]byte(`{"success":false,"error":"no custom providers"}`))
		return
	}

	var customProviders []providers.CustomProvider
	if err := json.Unmarshal([]byte(customProvidersJSON), &customProviders); err != nil {
		w.Write([]byte(`{"success":false,"error":"invalid data"}`))
		return
	}

	var newProviders []providers.CustomProvider
	for _, p := range customProviders {
		if p.URL != url {
			newProviders = append(newProviders, p)
		}
	}

	newJSON, _ := json.Marshal(newProviders)
	h.SettingsRepo.Set(repository.SettingCustomProviders, string(newJSON))

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
