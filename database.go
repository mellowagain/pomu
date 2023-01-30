package main

import (
	"database/sql"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"os"
)

func Connect() *sql.DB {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("failed to connect to database")
	}

	if err = db.Ping(); err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("failed to connect to database")
	}

	log.Info("successfully connected to database")
	return db
}
