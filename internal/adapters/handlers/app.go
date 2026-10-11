package handlers

import (
	"encoding/json"
	"net/http"

	"halloween/internal/core/services"
)

// GetGame returns whether the vote has started and finished, and the
// registration settings.
func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	st, err := h.game.State(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// PutGame lets the admin move the game to any state: waiting
// {started:false}, open {started:true, finished:false} or results {finished:true}.
func (h *Handler) PutGame(w http.ResponseWriter, r *http.Request) {
	var st services.GameState
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.game.Set(r.Context(), st); err != nil {
		writeServiceError(w, err)
		return
	}
	h.GetGame(w, r)
}
