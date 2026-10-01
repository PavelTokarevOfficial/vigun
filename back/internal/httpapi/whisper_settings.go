package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (a *API) listWhisperModels(w http.ResponseWriter, r *http.Request) {
	items, settings, err := a.whisperModels.List(r.Context())
	if err != nil {
		fail(w, 500, err)
		return
	}
	write(w, 200, map[string]any{"data": map[string]any{"models": items, "settings": settings}})
}

func (a *API) downloadWhisperModel(w http.ResponseWriter, r *http.Request) {
	if err := a.whisperModels.StartDownload(chi.URLParam(r, "id")); err != nil {
		fail(w, 422, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (a *API) selectWhisperModel(w http.ResponseWriter, r *http.Request) {
	if err := a.whisperModels.Select(r.Context(), chi.URLParam(r, "id")); err != nil {
		fail(w, 422, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) deleteWhisperModel(w http.ResponseWriter, r *http.Request) {
	if err := a.whisperModels.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		fail(w, 422, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) updateWhisperSettings(w http.ResponseWriter, r *http.Request) {
	current, err := a.whisperModels.Repo.Get(r.Context())
	if err != nil {
		fail(w, 500, err)
		return
	}
	input := current
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		fail(w, 400, errText("invalid JSON"))
		return
	}
	input.ActiveModel = current.ActiveModel
	if err = a.whisperModels.Update(r.Context(), input); err != nil {
		fail(w, 422, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
