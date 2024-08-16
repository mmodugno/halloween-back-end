package main

import (
	_ "github.com/go-sql-driver/mysql"
	"halloween/internal/adapters/handlers"
	"log"
	"net/http"
)

func main() {

	log.Println("server running in port ", 8080)
	log.Fatal(http.ListenAndServe(":8080", handlers.CreateRouter()))

}
