package main

import (
	"log"
	"net/http"
	"os"

	"auto-focus.app/cloud/handlers"
	"auto-focus.app/cloud/storage"
	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	storage, err := storage.NewSQLiteStorage("storage/data/autofocus.db")
	if err != nil {
		log.Fatal("Failed to create storage: ", err)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			log.Printf("Failed to close storage: %v", err)
		}
	}()

	srv := handlers.NewHttpServer(storage)

	log.Fatal(http.ListenAndServe(":8080", srv.Mux))
}
