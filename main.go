package main

import (
	"log"
	"net/http"

	"auto-focus.app/cloud/handlers"
	"auto-focus.app/cloud/storage"
)

func main() {
	db, err := storage.NewFileStorage("storage/data/customers.json")
	if err != nil {
		log.Fatal(err)
	}
	srv := handlers.NewHttpServer(db)

	log.Fatal(http.ListenAndServe(":8080", srv.Mux))
}
