package httpapp

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) StreamTrack(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")
	if trackID == "" {
		http.Error(w, "track ID required", http.StatusBadRequest)
		return
	}

	quality := r.URL.Query().Get("quality")
	if quality == "" {
		quality = h.Config.PlayQuality
		if quality == "" {
			quality = "HIGH"
		}
	}

	provider := h.ProviderManager.GetProvider()
	stream, mimeType, err := provider.GetStream(r.Context(), trackID, quality)
	if err != nil {
		h.Logger.Error("failed to get stream", "error", err, "trackID", trackID)
		http.Error(w, "failed to get stream", http.StatusInternalServerError)
		return
	}
	defer func() {
		if closeErr := stream.Close(); closeErr != nil {
			h.Logger.Error("stream close error", "error", closeErr)
		}
	}()

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Accept-Ranges", "none")
	w.Header().Set("Cache-Control", "no-cache")

	_, err = io.Copy(w, stream)
	if err != nil {
		h.Logger.Error("stream copy error", "error", err)
	}
}
