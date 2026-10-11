package main

import (
	"log"
	"net/http"
	"os"

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

	// Hosting platforms (e.g. Render) tell the app which port to use.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handlers.CreateRouter(h)))
}
