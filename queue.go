package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/lib/pq"
)

func (app *Application) GetQueue(w http.ResponseWriter, _ *http.Request) {
	tx, err := app.db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot start transaction", http.StatusInternalServerError)
		return
	}

	rows, err := tx.Query("select * from videos where finished = false order by start")

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to query for videos", http.StatusInternalServerError)
		return
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Println(err)
			sentry.CaptureException(err)
		}
	}(rows)

	var videos []Video

	for rows.Next() {
		var video Video

		if err := rows.Scan(&video.Id, pq.Array(&video.Submitters), &video.Start, &video.Finished); err != nil {
			sentry.CaptureException(err)
			log.Println(err)
			continue
		}

		videos = append(videos, video)
	}

	if err := tx.Commit(); err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")

	if err := json.NewEncoder(w).Encode(videos); err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot serialize to json", http.StatusInternalServerError)
	}
}
