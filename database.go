package main

import (
	"database/sql"
	"github.com/getsentry/sentry-go"
	_ "github.com/lib/pq"
	"log"
	"os"
)

func Connect() *sql.DB {
	url := os.Getenv("DATABASE_URL")

	db, err := sql.Open("postgres", url)

	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("Failed to open database: %s", err)
	}

	err = db.Ping()

	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("Failed to ping database: %s", err)
	}

	log.Println("Successfully connected to database")

	return db
}
