package handlers

import (
	"net/http"

	"github.com/cesargomez89/navidrums/internal/models"
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
	// Auto refresh
	jobs, _ := h.JobService.Repo.ListActiveJobs() // Use db directly for simplicity
	h.RenderFragment(w, "queue_list.html", jobs)
}

func (h *Handler) HistoryPage(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.JobService.Repo.ListFinishedJobs(20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.RenderPage(w, "history.html", jobs)
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
