package main

import (
	"log"
	"net/http"

	"halloween/internal/adapters/handlers"
	"halloween/internal/adapters/repository"
	"halloween/internal/core/services"
)

func main() {
	db, err := repository.Connect(repository.ConfigFromEnv())
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	h := handlers.New(services.NewUserService(db), services.NewVoteService(db), services.NewGameService(db))

	log.Println("server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", handlers.CreateRouter(h)))
}
