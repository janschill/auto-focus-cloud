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
	// Load environment file based on ENVIRONMENT variable
	envFile := ".env"
	if env := os.Getenv("ENVIRONMENT"); env == "staging" {
		envFile = ".env.staging"
	}

	// Load the appropriate environment file
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("Warning: Could not load %s file: %v", envFile, err)
		// Continue anyway in case env vars are set another way
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              os.Getenv("SENTRY_DSN"),
		TracesSampleRate: 1.0,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "storage/data/autofocus.db" // Default fallback
	}

	storage, err := storage.NewSQLiteStorage(databaseURL)
	if err != nil {
		log.Fatal("Failed to create storage: ", err)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			log.Printf("Failed to close storage: %v", err)
		}
	}()

	srv := handlers.NewHttpServer(storage)

	// Get port from environment variable, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Log startup information
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "production"
	}

	log.Printf("Starting Auto-Focus API server")
	log.Printf("Environment: %s", environment)
	log.Printf("Port: %s", port)
	log.Printf("Database: %s", databaseURL)
	log.Printf("Config file: %s", envFile)

	log.Fatal(http.ListenAndServe(":"+port, srv.Mux))
}
