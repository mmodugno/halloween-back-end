package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"halloween/internal/core/services"
	"halloween/internal/models"
)

// requireAdmin lets the request through only if the "User" header is the pass
// code of an admin.
func (h *Handler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := h.currentUser(w, r)
		if !ok {
			return
		}
		if !user.IsAdmin {
			writeError(w, http.StatusForbidden, services.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireAdminOrSetup is like requireAdmin, but while there are no users yet
// anyone can create them, so the first admin can be added.
func (h *Handler) requireAdminOrSetup(next http.Handler) http.Handler {
	admin := h.requireAdmin(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := h.users.Count(r.Context())
		if err != nil {
			writeServiceError(w, err)
			return
		}
		if n == 0 {
			next.ServeHTTP(w, r)
			return
		}
		admin.ServeHTTP(w, r)
	})
}

// AdminUsers lists every guest with their pass code.
func (h *Handler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.ListWithCodes(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user.ID = id
	if err := h.users.Update(r.Context(), user); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.users.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Response{Message: "user deleted", Code: http.StatusOK})
}

// ResetVotes deletes every vote and gives everyone `votes_per_user` votes.
func (h *Handler) ResetVotes(w http.ResponseWriter, r *http.Request) {
	var body struct {
		VotesPerUser int64 `json:"votes_per_user"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.VotesPerUser < 0 {
		writeError(w, http.StatusBadRequest, errors.New("votes_per_user must be 0 or more"))
		return
	}
	if err := h.votes.Reset(r.Context(), body.VotesPerUser); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Response{Message: "votes reset", Code: http.StatusOK})
}

func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid id"))
		return 0, false
	}
	return id, true
}
