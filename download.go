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
	"regexp"
	"strings"
)

const (
	TypeVideo     = "video"
	TypeFfmpegLog = "ffmpeg"
	TypeThumbnail = "thumbnail"
)

var crawlerUserAgentRegex = regexp.MustCompile("/bot|crawler|spider|crawling/i")

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

	if !finished && type_ != TypeThumbnail {
		http.Error(w, "video not yet finished", http.StatusBadRequest)
		return
	}

	increaseCount := r.Method != "HEAD" && !crawlerUserAgentRegex.MatchString(r.UserAgent())
	var url string

	switch type_ {
	case TypeVideo:
		if increaseCount {
			// update prometheus metrics
			videoDownloadCounter.Inc()

			// update our own per-video download counter, ignoring any errors if any occur
			_, _ = tx.Exec("update videos set downloads = downloads + 1 where id = $1", videoId)
		}

		url = fmt.Sprintf("%s/%s.mp4", os.Getenv("S3_DOWNLOAD_URL"), videoId)
		break
	case TypeFfmpegLog:
		if increaseCount {
			ffmpegLogDownloadCounter.Inc()
		}

		url = fmt.Sprintf("%s/%s.log", os.Getenv("S3_DOWNLOAD_URL"), videoId)
		break
	case TypeThumbnail:
		if increaseCount {
			thumbnailDownloadCounter.Inc()
		}

		url = thumbnail
		break
	}

	if err := tx.Commit(); err != nil {
		// only log the sentry error, don't actually exit early
		sentry.CaptureException(err)
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
