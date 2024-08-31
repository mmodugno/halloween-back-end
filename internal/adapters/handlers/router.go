package handlers

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"net/http"
)

type Response struct {
	Message string
	Code    int
}

var isFinished bool = false

func CreateRouter() *chi.Mux {
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
		r.Post("/users", PostUser)
		r.Get("/users/login", Login)
		r.Get("/users", GetAllUsers)
		r.Get("/users/passphrase", GetUserByPassphrase)
		r.Post("/votes", PostVote)
		r.Get("/results/winners", GetWinners)
		r.Get("/results", GetResults)
		r.Put("/finish", PutFinish)
		r.Get("/finish", GetFinish)
	})
	return r
}

func PutFinish(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	isFinished = true
	w.WriteHeader(http.StatusOK)
}

func GetFinish(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	type Response struct {
		Message bool
		Code    int
	}

	json.NewEncoder(w).Encode(Response{
		Message: isFinished,
		Code:    http.StatusOK,
	})
}
