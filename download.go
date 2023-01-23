package main

import (
	"database/sql"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"net/http"
	"os"
	"strings"
)

var videoDownloadCounter = promauto.NewCounter(prometheus.CounterOpts{
	Subsystem: "downloads",
	Name:      "video",
	Help:      "Number of video downloads (excludes HEAD requests)",
})

func (app *Application) VideoDownload(w http.ResponseWriter, r *http.Request) {
	userAgent := r.UserAgent()

	if len(strings.TrimSpace(userAgent)) == 0 || strings.Contains(userAgent, "Wget") ||
		strings.Contains(userAgent, "curl") || strings.Contains(userAgent, "Python-urllib") {
		http.Error(w, "automated requests only allowed with identify-able user agent (example: \"pomu (https://github.com/mellowagain/pomu)\")", http.StatusBadRequest)
		return
	}

	videoId := mux.Vars(r)["id"]

	tx, err := app.db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot start transaction", http.StatusInternalServerError)
		return
	}

	defer tx.Rollback()

	statement, err := tx.Prepare("select finished from videos where id = $1 limit 1")

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to prepare statement", http.StatusInternalServerError)
		return
	}

	var finished bool

	if err = statement.QueryRow(videoId).Scan(&finished); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "video not found", http.StatusNotFound)
		} else {
			sentry.CaptureException(err)
			http.Error(w, "failed to execute query", http.StatusInternalServerError)
		}

		return
	}

	if err := tx.Commit(); err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot commit transaction", http.StatusInternalServerError)
		return
	}

	if !finished {
		http.Error(w, "video not yet finished", http.StatusBadRequest)
		return
	}

	if r.Method != "HEAD" {
		videoDownloadCounter.Inc()
	}

	url := fmt.Sprintf("%s/%s.mp4", os.Getenv("S3_DOWNLOAD_URL"), videoId)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
