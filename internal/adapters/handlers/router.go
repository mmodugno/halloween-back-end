package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type Response struct {
	Message string
	Code    int
}

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
		r.Get("/users/login", Login)
		r.Post("/users", PostUser)
		r.Post("/users/batch", PostUsers)
		r.Get("/users", GetAllUsers)
		r.Get("/users/passphrase", GetUserByPassphrase)
		r.Post("/votes", PostVote)
		r.Get("/results/winners", GetWinners)
		r.Get("/results", GetResults)
		r.Put("/finish", PutFinish)
		r.Put("/finish/cancel", CancelFinish)
		r.Get("/finish", GetFinish)
		r.Put("/start", StartGame)
		r.Get("/start", IsStarted)
		r.Post("/mocks/create", TestsBach)
	})
	return r
}
