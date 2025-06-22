package main

import (
	"log"
	"net/http"

	"auto-focus.app/cloud/handlers"
	"auto-focus.app/cloud/storage"
)

func main() {
	storage, err := storage.NewSQLiteStorage("storage/data/autofocus.db")
	if err != nil {
		log.Fatal("Failed to create storage: ", err)
	}
	defer storage.Close()

	srv := handlers.NewHttpServer(storage)

	log.Fatal(http.ListenAndServe(":8080", srv.Mux))
}
