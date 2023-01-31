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

const (
	TypeVideo     = "video"
	TypeFfmpegLog = "ffmpeg"
	TypeThumbnail = "thumbnail"
)

var videoDownloadCounter = promauto.NewCounter(prometheus.CounterOpts{
	Subsystem: "downloads",
	Name:      "video",
	Help:      "Number of video downloads (excludes HEAD requests)",
})

var ffmpegLogDownloadCounter = promauto.NewCounter(prometheus.CounterOpts{
	Subsystem: "downloads",
	Name:      "log",
	Help:      "Number of ffmpeg log requests (excludes HEAD)",
})

var thumbnailDownloadCounter = promauto.NewCounter(prometheus.CounterOpts{
	Subsystem: "downloads",
	Name:      "thumbnail",
	Help:      "Number of thumbnail requests (excludes HEAD)",
})

func (app *Application) VideoDownload(w http.ResponseWriter, r *http.Request) {
	userAgent := r.UserAgent()

	if len(strings.TrimSpace(userAgent)) == 0 || strings.Contains(userAgent, "Wget") ||
		strings.Contains(userAgent, "curl") || strings.Contains(userAgent, "Python-urllib") {
		http.Error(w, "automated requests only allowed with identify-able user agent (example: \"pomu (https://github.com/mellowagain/pomu)\")", http.StatusBadRequest)
		return
	}

	variables := mux.Vars(r)
	videoId := variables["id"]
	type_ := variables["type"]

	if type_ != TypeVideo && type_ != TypeFfmpegLog && type_ != TypeThumbnail {
		http.Error(w, fmt.Sprintf("unknown type \"%s\"", type_), http.StatusNotFound)
		return
	}

	tx, err := app.db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot start transaction", http.StatusInternalServerError)
		return
	}

	defer tx.Rollback()

	statement, err := tx.Prepare("select finished, thumbnail from videos where id = $1 limit 1")

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to prepare statement", http.StatusInternalServerError)
		return
	}

	var finished bool
	var thumbnail string

	if err = statement.QueryRow(videoId).Scan(&finished, &thumbnail); err != nil {
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

	if !finished && type_ != TypeThumbnail {
		http.Error(w, "video not yet finished", http.StatusBadRequest)
		return
	}

	var url string

	switch type_ {
	case TypeVideo:
		if r.Method != "HEAD" {
			videoDownloadCounter.Inc()
		}

		url = fmt.Sprintf("%s/%s.mp4", os.Getenv("S3_DOWNLOAD_URL"), videoId)
		break
	case TypeFfmpegLog:
		if r.Method != "HEAD" {
			ffmpegLogDownloadCounter.Inc()
		}

		url = fmt.Sprintf("%s/%s.log", os.Getenv("S3_DOWNLOAD_URL"), videoId)
		break
	case TypeThumbnail:
		if r.Method != "HEAD" {
			thumbnailDownloadCounter.Inc()
		}

		url = thumbnail
		break
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
