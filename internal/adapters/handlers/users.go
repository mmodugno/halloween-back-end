package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"halloween/internal/models"
)

var errMissingPassphrase = errors.New("passphrase is required")

// PostUsers creates several users and returns them with their generated pass codes.
func (h *Handler) PostUsers(w http.ResponseWriter, r *http.Request) {
	var users []models.User
	if err := json.NewDecoder(r.Body).Decode(&users); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	h.createUsers(w, r, users)
}

func (h *Handler) createUsers(w http.ResponseWriter, r *http.Request, users []models.User) {
	created, err := h.users.Create(r.Context(), users)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Response{
		Message: fmt.Sprintf("created %d users successfully", len(created)),
		Code:    http.StatusCreated,
		Users:   created,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, models.LoggedUser{
		ID:           user.ID,
		IsAdmin:      user.IsAdmin,
		PendingVotes: user.PendingVotes,
		CostumeID:    user.CostumeID,
	})
}

func (h *Handler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// currentUser looks up the user whose pass code is in the "User" header.
// On failure it writes the error response and returns false.
func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	pw := r.Header.Get("User")
	if pw == "" {
		writeError(w, http.StatusUnauthorized, errMissingPassphrase)
		return nil, false
	}
	user, err := h.users.GetByPassphrase(r.Context(), pw)
	if err != nil {
		writeServiceError(w, err)
		return nil, false
	}
	return user, true
}
