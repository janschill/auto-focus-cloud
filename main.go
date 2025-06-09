package main

import (
	"log"
	"net/http"

	"auto-focus.app/cloud/handlers"
	"auto-focus.app/cloud/models"
	"auto-focus.app/cloud/storage"
)

func main() {
	db := storage.Database{
		"1": models.Customer{
			Id:    "1",
			Email: "john@example.com",
			Licenses: []models.License{
				{Key: "foo", Version: "1.0.0"},
			},
		},
	}
	srv := handlers.NewHttpServer(db)

	log.Fatal(http.ListenAndServe(":8080", srv.Mux))
}
