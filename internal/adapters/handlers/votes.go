package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"halloween/internal/models"
)

func (h *Handler) PostVote(w http.ResponseWriter, r *http.Request) {
	var vote models.Vote
	if err := json.NewDecoder(r.Body).Decode(&vote); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if vote.CostumeID == 0 {
		writeError(w, http.StatusBadRequest, errors.New("costume_id is required"))
		return
	}
	passphrase := r.Header.Get("User")
	if passphrase == "" {
		writeError(w, http.StatusUnauthorized, errMissingPassphrase)
		return
	}

	if err := h.votes.Cast(r.Context(), passphrase, vote.CostumeID, vote.Message); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Response{Message: "vote created successfully", Code: http.StatusCreated})
}

func (h *Handler) GetWinners(w http.ResponseWriter, r *http.Request) {
	winners, err := h.votes.Winners(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, winners)
}

func (h *Handler) GetResults(w http.ResponseWriter, r *http.Request) {
	results, err := h.votes.Results(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// GetMessages returns who voted for a costume and what they wrote. Only the
// people wearing it (or an admin) can read them.
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if !user.IsAdmin && user.CostumeID != id {
		writeError(w, http.StatusForbidden, errors.New("you can only read your own messages"))
		return
	}
	messages, err := h.votes.Messages(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

// UndoVote takes back a vote cast a few seconds ago (see services.UndoWindow).
func (h *Handler) UndoVote(w http.ResponseWriter, r *http.Request) {
	candidateID, err := strconv.ParseInt(chi.URLParam(r, "candidateID"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid costume id"))
		return
	}
	passphrase := r.Header.Get("User")
	if passphrase == "" {
		writeError(w, http.StatusUnauthorized, errMissingPassphrase)
		return
	}
	if err := h.votes.Undo(r.Context(), passphrase, candidateID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Response{Message: "vote removed", Code: http.StatusOK})
}

const feedSize = 30

// GetFeed returns the latest anonymous messages for the live board.
func (h *Handler) GetFeed(w http.ResponseWriter, r *http.Request) {
	items, err := h.votes.Feed(r.Context(), feedSize)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// GetStats returns public totals (guests, votes cast, votes possible).
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	st, err := h.votes.Stats(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}
