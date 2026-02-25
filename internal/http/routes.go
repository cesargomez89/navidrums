package httpapp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/cesargomez89/navidrums/internal/catalog"
	"github.com/cesargomez89/navidrums/internal/constants"
	"github.com/cesargomez89/navidrums/internal/domain"
	"github.com/cesargomez89/navidrums/internal/http/dto"
	"github.com/cesargomez89/navidrums/internal/musicbrainz"
	"github.com/cesargomez89/navidrums/internal/store"
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
		_, _ = w.Write([]byte(""))
		return
	}

	provider := h.ProviderManager.GetProvider()

	results, err := provider.Search(r.Context(), query, searchType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.RenderFragment(w, "search_results.html", results)
}

func (h *Handler) ArtistPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	artist, err := h.ProviderManager.GetProvider().GetArtist(r.Context(), id)
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
	album, err := h.ProviderManager.GetProvider().GetAlbum(r.Context(), id)
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
	pl, err := h.ProviderManager.GetProvider().GetPlaylist(r.Context(), id)
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
	_, _ = w.Write([]byte("<div class='alert alert-success'>Download started!</div>"))
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
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}

	jobs, total, err := h.JobService.ListActiveJobs(page, constants.MaxSearchResults)
	if err != nil {
		h.Logger.Error("Failed to list active jobs", "error", err)
	}

	pagination := dto.NewPagination(page, constants.MaxSearchResults, total, "/htmx/queue/active", "#tab-content", "")

	h.RenderFragment(w, "components/active_tab.html", map[string]interface{}{
		"ActiveJobs": jobs,
		"Pagination": pagination,
	})
}

func (h *Handler) QueueHistoryHTMX(w http.ResponseWriter, r *http.Request) {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}

	jobs, total, err := h.JobService.ListFinishedJobs(page, constants.MaxHistoryItems)
	if err != nil {
		h.Logger.Error("Failed to list finished jobs", "error", err)
		return
	}

	stats, err := h.JobService.GetJobStats()
	if err != nil {
		h.Logger.Error("Failed to get job stats", "error", err)
	}

	pagination := dto.NewPagination(page, constants.MaxHistoryItems, total, "/htmx/queue/history", "#tab-content", "")

	h.RenderFragment(w, "components/history_tab.html", map[string]interface{}{
		"HistoryJobs": jobs,
		"Stats":       stats,
		"Pagination":  pagination,
	})
}

func (h *Handler) CancelJobHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.JobService.CancelJob(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jobs, _, err := h.JobService.ListActiveJobs(1, constants.MaxSearchResults)
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

	jobs, _, err := h.JobService.ListActiveJobs(1, constants.MaxSearchResults)
	if err != nil {
		h.Logger.Error("Failed to list active jobs", "error", err)
	}
	h.RenderFragment(w, "components/active_tab.html", map[string]interface{}{
		"ActiveJobs": jobs,
	})
}

func (h *Handler) GetProvidersHTMX(w http.ResponseWriter, r *http.Request) {
	active := h.ProviderManager.GetBaseURL()
	defaultURL := h.ProviderManager.GetDefaultURL()

	var customProviders []catalog.CustomProvider
	customProvidersJSON, err := h.SettingsRepo.Get(store.SettingCustomProviders)
	if err == nil && customProvidersJSON != "" {
		if unmarshalErr := json.Unmarshal([]byte(customProvidersJSON), &customProviders); unmarshalErr != nil {
			h.Logger.Error("Failed to unmarshal custom providers", "error", unmarshalErr)
		}
	}

	response := map[string]interface{}{
		"predefined": json.RawMessage(catalog.GetPredefinedProvidersJSON()),
		"custom":     customProviders,
		"active":     active,
		"default":    defaultURL,
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

	_, _ = w.Write([]byte(`{"success":true,"url":"` + url + `"}`))
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
		if unmarshalErr := json.Unmarshal([]byte(customProvidersJSON), &customProviders); unmarshalErr != nil {
			h.Logger.Error("Failed to unmarshal custom providers", "error", unmarshalErr)
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

	_, _ = w.Write([]byte(`{"success":true}`))
}

func (h *Handler) RemoveCustomProviderHTMX(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	customProvidersJSON, err := h.SettingsRepo.Get(store.SettingCustomProviders)
	if err != nil || customProvidersJSON == "" {
		_, _ = w.Write([]byte(`{"success":false,"error":"no custom catalog"}`))
		return
	}

	var customProviders []catalog.CustomProvider
	if unmarshalErr := json.Unmarshal([]byte(customProvidersJSON), &customProviders); unmarshalErr != nil {
		_, _ = w.Write([]byte(`{"success":false,"error":"invalid data"}`))
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

	_, _ = w.Write([]byte(`{"success":true}`))
}

func (h *Handler) GetGenreMapHTMX(w http.ResponseWriter, r *http.Request) {
	customMapJSON, err := h.SettingsRepo.Get(store.SettingGenreMap)
	if err != nil {
		h.Logger.Error("Failed to get genre map", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"default": musicbrainz.DefaultGenreMap,
		"custom":  nil,
	}

	if customMapJSON != "" {
		var customMap map[string]string
		if unmarshalErr := json.Unmarshal([]byte(customMapJSON), &customMap); unmarshalErr != nil {
			h.Logger.Error("Failed to unmarshal custom genre map", "error", unmarshalErr)
		} else {
			response["custom"] = customMap
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Logger.Error("Failed to encode genre map response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handler) SetGenreMapHTMX(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GenreMap map[string]string `json:"genreMap"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	genreMapJSON, err := json.Marshal(req.GenreMap)
	if err != nil {
		h.Logger.Error("Failed to marshal genre map", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.SettingsRepo.Set(store.SettingGenreMap, string(genreMapJSON)); err != nil {
		h.Logger.Error("Failed to save genre map", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write([]byte(`{"success":true}`))
}

func (h *Handler) ResetGenreMapHTMX(w http.ResponseWriter, r *http.Request) {
	if err := h.SettingsRepo.Delete(store.SettingGenreMap); err != nil {
		h.Logger.Error("Failed to reset genre map", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write([]byte(`{"success":true}`))
}

func (h *Handler) SimilarAlbumsHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	albums, err := h.ProviderManager.GetProvider().GetSimilarAlbums(r.Context(), id)
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
	filter := r.URL.Query().Get("filter")

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}

	var tracks []*domain.Track
	var total int
	var err error

	extraParams := ""

	switch {
	case query != "":
		tracks, total, err = h.DownloadsService.SearchDownloads(query, page, constants.MaxSearchResults)
		extraParams = "q=" + query
	case filter != "":
		tracks, total, err = h.DownloadsService.FilterDownloads(filter, page, constants.MaxSearchResults)
		extraParams = "filter=" + filter
	default:
		tracks, total, err = h.DownloadsService.ListDownloads(page, constants.MaxSearchResults)
	}
	if err != nil {
		h.Logger.Error("Failed to list downloads", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	pagination := dto.NewPagination(page, constants.MaxSearchResults, total, "/htmx/downloads", "#downloads-container", extraParams)

	h.RenderFragment(w, "components/downloads_list.html", map[string]interface{}{
		"Downloads":  tracks,
		"Filter":     filter,
		"Pagination": pagination,
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

func (h *Handler) BulkDeleteHTMX(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	ids := r.Form["ids[]"]
	for _, id := range ids {
		if err := h.DownloadsService.DeleteDownload(id); err != nil {
			h.Logger.Error("Failed to delete download", "id", id, "error", err)
		}
	}

	h.DownloadsHTMX(w, r)
}

func (h *Handler) BulkSyncHTMX(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	ids := r.Form["ids[]"]
	count := 0
	for _, id := range ids {
		if err := h.DownloadsService.EnqueueSyncMetadataJob(id); err != nil {
			h.Logger.Error("Failed to enqueue sync job", "id", id, "error", err)
			continue
		}
		count++
	}

	var tracks []*domain.Track
	query := r.URL.Query().Get("q")
	if query != "" {
		tracks, _, _ = h.DownloadsService.SearchDownloads(query, 1, constants.MaxSearchResults)
	} else {
		tracks, _, _ = h.DownloadsService.ListDownloads(1, constants.MaxSearchResults)
	}

	h.RenderFragment(w, "components/downloads_list.html", map[string]interface{}{
		"Downloads":    tracks,
		"SyncEnqueued": count,
	})
}

func (h *Handler) BulkUpdateGenreHTMX(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	ids := r.Form["ids[]"]
	genre := r.FormValue("genre")
	if genre == "" {
		http.Error(w, "genre is required", http.StatusBadRequest)
		return
	}

	for _, providerID := range ids {
		track, err := h.DownloadsService.GetDownloadByProviderID(providerID)
		if err != nil || track == nil {
			h.Logger.Error("Failed to get track for genre update", "provider_id", providerID, "error", err)
			continue
		}

		updates := map[string]interface{}{"genre": genre}

		if err := h.DownloadsService.UpdateTrackPartial(track.ID, updates); err != nil {
			h.Logger.Error("Failed to update genre", "track_id", track.ID, "error", err)
			continue
		}

		if err := h.DownloadsService.EnqueueSyncFileJob(providerID); err != nil {
			h.Logger.Error("Failed to enqueue sync job", "provider_id", providerID, "error", err)
		}
	}

	h.DownloadsHTMX(w, r)
}

func (h *Handler) TrackPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var trackID int
	if _, err := fmt.Sscanf(id, "%d", &trackID); err != nil {
		http.Error(w, "Invalid track ID", http.StatusBadRequest)
		return
	}

	track, err := h.DownloadsService.GetTrackByID(trackID)
	if err != nil {
		h.Logger.Error("Failed to get track", "error", err)
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	h.RenderPage(w, "track.html", map[string]interface{}{
		"ActivePage": "downloads",
		"Track":      track,
	})
}

func (h *Handler) TrackHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var trackID int
	if _, err := fmt.Sscanf(id, "%d", &trackID); err != nil {
		http.Error(w, "Invalid track ID", http.StatusBadRequest)
		return
	}

	track, err := h.DownloadsService.GetTrackByID(trackID)
	if err != nil {
		h.Logger.Error("Failed to get track", "error", err)
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	h.RenderFragment(w, "components/track_form.html", map[string]interface{}{
		"Track": track,
	})
}

func (h *Handler) SaveTrackHTMX(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var trackID int
	if _, err := fmt.Sscanf(id, "%d", &trackID); err != nil {
		http.Error(w, "Invalid track ID", http.StatusBadRequest)
		return
	}

	track, err := h.DownloadsService.GetTrackByID(trackID)
	if err != nil {
		h.Logger.Error("Failed to get track", "error", err)
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	if parseErr := r.ParseForm(); parseErr != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	var d dto.TrackUpdateRequest
	if decodeErr := h.FormDecoder.Decode(&d, r.PostForm); decodeErr != nil {
		h.Logger.Error("Failed to decode form", "error", decodeErr)
		http.Error(w, "Failed to decode form", http.StatusBadRequest)
		return
	}

	validationErrs := d.Validate()
	if len(validationErrs) > 0 {
		h.Logger.Warn("Track validation failed", "errors", validationErrs)
		h.RenderFragment(w, "components/track_form.html", map[string]interface{}{
			"Track":            track,
			"ValidationErrors": dto.ToMap(validationErrs),
		})
		return
	}

	updates := d.ToUpdates()
	if len(updates) == 0 {
		h.RenderFragment(w, "components/track_form.html", map[string]interface{}{
			"Track": track,
		})
		return
	}

	if updateErr := h.DownloadsService.UpdateTrackPartial(trackID, updates); updateErr != nil {
		h.Logger.Error("Failed to update track", "error", updateErr)
		http.Error(w, updateErr.Error(), http.StatusInternalServerError)
		return
	}

	track, err = h.DownloadsService.GetTrackByID(trackID)
	if err != nil {
		h.Logger.Error("Failed to get track", "error", err)
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	h.RenderFragment(w, "components/track_form.html", map[string]interface{}{
		"Track":       track,
		"SaveSuccess": true,
	})
}

type enrichAction string

const (
	enrichActionSyncFile        enrichAction = "sync_file"
	enrichActionSyncMusicBrainz enrichAction = "sync_musicbrainz"
	enrichActionSyncHiFi        enrichAction = "sync_hifi"
)

func (h *Handler) handleTrackEnrich(w http.ResponseWriter, r *http.Request) (*domain.Track, bool) {
	id := chi.URLParam(r, "id")
	var trackID int
	if _, err := fmt.Sscanf(id, "%d", &trackID); err != nil {
		http.Error(w, "Invalid track ID", http.StatusBadRequest)
		return nil, false
	}

	track, err := h.DownloadsService.GetTrackByID(trackID)
	if err != nil {
		h.Logger.Error("Failed to get track", "error", err)
		http.Error(w, "Track not found", http.StatusNotFound)
		return nil, false
	}

	if parseErr := r.ParseForm(); parseErr != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return nil, false
	}

	var d dto.TrackUpdateRequest
	if decodeErr := h.FormDecoder.Decode(&d, r.PostForm); decodeErr != nil {
		h.Logger.Error("Failed to decode form", "error", decodeErr)
		http.Error(w, "Failed to decode form", http.StatusBadRequest)
		return nil, false
	}

	validationErrs := d.Validate()
	if len(validationErrs) > 0 {
		h.Logger.Warn("Track validation failed", "errors", validationErrs)
		h.RenderFragment(w, "components/track_form.html", map[string]interface{}{
			"Track":            track,
			"ValidationErrors": dto.ToMap(validationErrs),
		})
		return nil, false
	}

	updates := d.ToUpdates()
	if len(updates) > 0 {
		if updateErr := h.DownloadsService.UpdateTrackPartial(trackID, updates); updateErr != nil {
			h.Logger.Error("Failed to update track", "error", updateErr)
			http.Error(w, updateErr.Error(), http.StatusInternalServerError)
			return nil, false
		}
	}

	track, _ = h.DownloadsService.GetTrackByID(trackID)
	return track, true
}

func (h *Handler) renderEnrichResponse(w http.ResponseWriter, track *domain.Track, action enrichAction) {
	h.RenderFragment(w, "components/track_form.html", map[string]interface{}{
		"Track":           track,
		"JobEnqueued":     true,
		"JobEnqueuedType": string(action),
	})
}

func (h *Handler) SyncTrackHTMX(w http.ResponseWriter, r *http.Request) {
	track, ok := h.handleTrackEnrich(w, r)
	if !ok {
		return
	}

	if err := h.DownloadsService.EnqueueSyncFileJob(track.ProviderID); err != nil {
		h.Logger.Error("Failed to enqueue sync job", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderEnrichResponse(w, track, enrichActionSyncFile)
}

func (h *Handler) EnrichTrackHTMX(w http.ResponseWriter, r *http.Request) {
	track, ok := h.handleTrackEnrich(w, r)
	if !ok {
		return
	}

	if err := h.DownloadsService.EnqueueSyncMetadataJob(track.ProviderID); err != nil {
		h.Logger.Error("Failed to enqueue enrich job", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderEnrichResponse(w, track, enrichActionSyncMusicBrainz)
}

func (h *Handler) EnrichHiFiHTMX(w http.ResponseWriter, r *http.Request) {
	track, ok := h.handleTrackEnrich(w, r)
	if !ok {
		return
	}

	if err := h.DownloadsService.EnqueueSyncHiFiJob(track.ProviderID); err != nil {
		h.Logger.Error("Failed to enqueue enrich Hi-Fi job", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderEnrichResponse(w, track, enrichActionSyncHiFi)
}

func (h *Handler) SyncAllHTMX(w http.ResponseWriter, r *http.Request) {
	count, err := h.DownloadsService.EnqueueSyncJobs()
	if err != nil {
		h.Logger.Error("Failed to enqueue sync jobs", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tracks, _, _ := h.DownloadsService.ListDownloads(1, constants.MaxSearchResults)
	h.RenderFragment(w, "components/downloads_list.html", map[string]interface{}{
		"Downloads":    tracks,
		"SyncEnqueued": count,
	})
}
