package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"halloween/internal/core/services"
	"halloween/internal/models"
)

type Handler struct {
	users *services.UserService
	votes *services.VoteService
	game  *services.GameService
}

func New(users *services.UserService, votes *services.VoteService, game *services.GameService) *Handler {
	return &Handler{users: users, votes: votes, game: game}
}

type Response struct {
	Message string
	Code    int
	Users   []models.User `json:",omitempty"`
}

type ResponseBool struct {
	Message bool
	Code    int
}

func CreateRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"X-PINGOTHER", "Accept", "Authorization", "Content-Type", "X-CRSF-Token", "User"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api", func(r chi.Router) {
		r.Get("/users/login", h.Login)
		r.Post("/users", h.PostUser)
		r.Post("/users/batch", h.PostUsers)
		r.Get("/users", h.GetAllUsers)
		r.Get("/users/passphrase", h.GetUserByPassphrase)
		r.Post("/votes", h.PostVote)
		r.Delete("/votes/{candidateID}", h.UndoVote)
		r.Get("/feed", h.GetFeed)
		r.Get("/results", h.GetResults)
		r.Get("/results/winners", h.GetWinners)
		r.Get("/results/{id}", h.GetMessages)
		r.Put("/finish", h.setState(setFinished, true))
		r.Put("/finish/cancel", h.setState(setFinished, false))
		r.Get("/finish", h.GetFinished)
		r.Put("/start", h.setState(setStarted, true))
		r.Get("/start", h.GetStarted)
		r.Post("/mocks/create", h.LoadMocks)
	})
	return r
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, Response{Message: "error: " + err.Error(), Code: status})
}

// writeServiceError maps the services' errors to HTTP status codes.
func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, services.ErrNotFound):
		status = http.StatusUnauthorized
	case errors.Is(err, services.ErrSelfVote), errors.Is(err, services.ErrUnknownCandidate):
		status = http.StatusBadRequest
	case errors.Is(err, services.ErrVoteNotFound):
		status = http.StatusNotFound
	case errors.Is(err, services.ErrNoVotesLeft), errors.Is(err, services.ErrAlreadyVotedFor),
		errors.Is(err, services.ErrVotingClosed), errors.Is(err, services.ErrUndoExpired):
		status = http.StatusConflict
	}
	writeError(w, status, err)
}
