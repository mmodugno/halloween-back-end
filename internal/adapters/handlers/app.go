package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	"halloween/internal/core/services"
	"halloween/internal/models"
)

// GetGame returns whether the vote has started and finished.
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
	writeJSON(w, http.StatusOK, st)
}

func (h *Handler) GetStarted(w http.ResponseWriter, r *http.Request) {
	h.writeState(w, r, func(started, _ bool) bool { return started })
}

func (h *Handler) GetFinished(w http.ResponseWriter, r *http.Request) {
	h.writeState(w, r, func(_, finished bool) bool { return finished })
}

func (h *Handler) writeState(w http.ResponseWriter, r *http.Request, pick func(started, finished bool) bool) {
	st, err := h.game.State(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ResponseBool{Message: pick(st.Started, st.Finished), Code: http.StatusOK})
}

// setState returns a handler that stores a flag and answers with its new value.
func (h *Handler) setState(set func(*Handler, *http.Request, bool) error, value bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := set(h, r, value); err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ResponseBool{Message: value, Code: http.StatusOK})
	}
}

func setStarted(h *Handler, r *http.Request, v bool) error { return h.game.SetStarted(r.Context(), v) }
func setFinished(h *Handler, r *http.Request, v bool) error {
	return h.game.SetFinished(r.Context(), v)
}

// LoadMocks fills the DB with the users and votes in mock.json.
// Each entry is read both as a user and as a vote.
func (h *Handler) LoadMocks(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("mock.json")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	var users []models.User
	var votes []models.Vote
	if err := json.Unmarshal(data, &users); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := json.Unmarshal(data, &votes); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if _, err := h.users.Create(r.Context(), users, true); err != nil {
		writeServiceError(w, err)
		return
	}
	if err := h.votes.Seed(r.Context(), votes); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Response{Message: "mocks created", Code: http.StatusOK})
}
