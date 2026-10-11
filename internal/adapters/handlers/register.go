package handlers

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"halloween/internal/core/services"
	"halloween/internal/models"
)

// GetCostumes lists the costumes people can vote for (or join when signing
// up), with who is wearing each one.
func (h *Handler) GetCostumes(w http.ResponseWriter, r *http.Request) {
	costumes, err := h.users.Costumes(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, costumes)
}

type registerResponse struct {
	Costume models.Costume `json:"costume"`
	Users   []models.User  `json:"users"`
}

// Register is the public sign-up form: anyone with the link can add their
// costume, themselves and their costume partners before the vote starts.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var reg services.Registration
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&reg); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	people := 1 + len(reg.Partners)
	if !h.signups.allow(clientIP(r), people) {
		writeError(w, http.StatusTooManyRequests, errTooManySignups)
		return
	}
	costume, users, err := h.users.Register(r.Context(), reg)
	if err != nil {
		h.signups.refund(clientIP(r), people)
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, registerResponse{Costume: costume, Users: users})
}

// PutRegistration lets the admin open or close sign-ups and set how many
// votes new guests get: {"registration_open": true, "votes_per_user": 2}.
func (h *Handler) PutRegistration(w http.ResponseWriter, r *http.Request) {
	var body services.GameState
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.VotesPerUser < 0 {
		writeError(w, http.StatusBadRequest, errors.New("votes_per_user must be 0 or more"))
		return
	}
	if err := h.game.SetRegistration(r.Context(), body.RegistrationOpen, body.VotesPerUser); err != nil {
		writeServiceError(w, err)
		return
	}
	h.GetGame(w, r)
}

// RenameCostume changes a costume's name: {"name": "Cazafantasmas"}.
func (h *Handler) RenameCostume(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.users.RenameCostume(r.Context(), id, body.Name); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

var errTooManySignups = errors.New("too many sign-ups from this connection, try again in a few minutes")

// signupLimiter caps how many people one IP can sign up per window, so a
// script can't flood the guest list. It's generous because everyone at the
// party may share the same wifi.
type signupLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

func newSignupLimiter(limit int, window time.Duration) *signupLimiter {
	return &signupLimiter{limit: limit, window: window, hits: map[string][]time.Time{}}
}

func (l *signupLimiter) allow(ip string, people int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	recent := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if now.Sub(t) < l.window {
			recent = append(recent, t)
		}
	}
	if len(recent)+people > l.limit {
		l.hits[ip] = recent
		return false
	}
	for i := 0; i < people; i++ {
		recent = append(recent, now)
	}
	l.hits[ip] = recent
	return true
}

// refund gives back the slots of a sign-up that failed (e.g. a typo), so
// fixing it doesn't count twice.
func (l *signupLimiter) refund(ip string, people int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if n := len(l.hits[ip]) - people; n >= 0 {
		l.hits[ip] = l.hits[ip][:n]
	}
}

// clientIP is the caller's address; behind Render's proxy it's the first
// entry of X-Forwarded-For.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
