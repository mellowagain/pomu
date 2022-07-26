package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/lib/pq"
	"golang.org/x/exp/slices"
	"golang.org/x/oauth2"
)

type Video struct {
	Id         string    `json:"id"`
	Submitters []string  `json:"submitters"`
	Start      time.Time `json:"scheduledStart"`
	Finished   bool      `json:"finished"`
}

type VideoRequest struct {
	VideoUrl string `json:"videoUrl"`
	Quality  int32  `json:"quality"`
}

func (r *VideoRequest) Id() (string, error) {
	parsed, err := url.Parse(r.VideoUrl)
	if err != nil {
		return "", err
	}
	return parsed.Query().Get("v"), nil
}

func (app *Application) SubmitVideo(w http.ResponseWriter, r *http.Request) {
	var request VideoRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "failed to decode json request body", http.StatusInternalServerError)
		return
	}

	cookie, err := r.Cookie("pomu")

	if err != nil {
		http.Error(w, "please login first", http.StatusUnauthorized)
		return
	}

	var token *oauth2.Token

	if err = app.secureCookie.Decode("oauthToken", cookie.Value, &token); err != nil {
		http.Error(w, "please login again", http.StatusUnauthorized)
		return
	}

	user, err := ResolveUser(token, app.db)

	if err != nil {
		http.Error(w, "failed to resolve user", http.StatusUnauthorized)
		return
	}

	videoId := ParseVideoID(request.VideoUrl)
	videoMetadata, err := GetVideoMetadata(videoId, token)

	if !IsLivestream(videoMetadata) {
		http.Error(w, "can only archive livestreams (for videos use youtube-dl)", http.StatusBadRequest)
		log.Println("Ignoring submission", videoId, " as it is not a livestream")
		return
	}

	if IsLivestreamEnded(videoMetadata) {
		http.Error(w, "can only archive livestreams in the future or currently running (try youtube-dl)", http.StatusBadRequest)
		log.Println("Ignoring submission", videoId, " as it has ended")
		return
	}

	var startTime time.Time

	if IsLivestreamStarted(videoMetadata) {
		startTime, err = time.Parse(time.RFC3339, videoMetadata.LiveStreamingDetails.ActualStartTime)
	} else {
		startTime, err = time.Parse(time.RFC3339, videoMetadata.LiveStreamingDetails.ScheduledStartTime)
	}

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to parse start time", http.StatusInternalServerError)
		return
	}

	tx, err := app.db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}

	var video Video
	err = tx.QueryRow("select * from videos where id = $1 limit 1", videoId).Scan(
		&video.Id, pq.Array(&video.Submitters), &video.Start, &video.Finished)

	var reschedule bool

	if err != nil {
		if err != sql.ErrNoRows {
			sentry.CaptureException(err)
			http.Error(w, "failed to check if video already is being archived", http.StatusInternalServerError)
			return
		}

		statement, err := tx.Prepare("insert into videos (id, submitters, start) values ($1, $2, $3) returning *")

		if err != nil {
			sentry.CaptureException(err)
			http.Error(w, "failed to prepare statement", http.StatusInternalServerError)
			return
		}
		row := statement.QueryRow(videoId, pq.Array([]string{user.id}), startTime)

		if err := row.Err(); err != nil {
			sentry.CaptureException(err)
			http.Error(w, "failed to create new video", http.StatusInternalServerError)
			return
		}

		if row.Scan(&video.Id, pq.Array(&video.Submitters), &video.Start, &video.Finished) != nil {
			sentry.CaptureException(err)
			http.Error(w, "failed to create new video", http.StatusInternalServerError)
			return
		}

		reschedule = true
	} else {
		if !slices.Contains(video.Submitters, user.id) {
			statement, err := tx.Prepare("update videos set submitters = array_append(submitters, $1), start = $2 where $3 returning *")

			if err != nil {
				sentry.CaptureException(err)
				http.Error(w, "failed to prepare statement", http.StatusInternalServerError)
				return
			}

			if err := statement.QueryRow(user.id, startTime, video.Id).Scan(&video.Id, &video.Submitters, &video.Start, &video.Finished); err != nil {
				sentry.CaptureException(err)
				http.Error(w, "failed to update existing video", http.StatusInternalServerError)
				return
			}
		}

		reschedule = false
	}

	log.Printf("New video submitted: %s (quality %d)\n", request.VideoUrl, request.Quality)

	if reschedule {
		if IsLivestreamStarted(videoMetadata) {
			if _, err := Scheduler.SingletonMode().Every("10s").LimitRunsTo(1).Tag(videoId).StartImmediately().Do(StartRecording, app.db, request); err != nil {
				sentry.CaptureException(err)
				http.Error(w, "failed to schedule and start recording job", http.StatusInternalServerError)
				return
			}

			log.Printf("Livestream already started, starting recording immediatly")
		} else {
			if _, err := Scheduler.SingletonMode().LimitRunsTo(1).StartAt(startTime).Tag(videoId).Do(StartRecording, app.db, request); err != nil {
				sentry.CaptureException(err)
				http.Error(w, "failed to schedule recording job", http.StatusInternalServerError)
				return
			}

			log.Printf("Livestream recording scheduled for %s", startTime.Format(time.RFC3339))
		}
	}

	if err := tx.Commit(); err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Expires", strings.ReplaceAll(startTime.UTC().Format(time.RFC1123), "UTC", "GMT"))

	if err := json.NewEncoder(w).Encode(video); err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot serialize to json", http.StatusInternalServerError)
	}
}

func PeekForQualities(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")

	if len(url) <= 0 {
		http.Error(w, "required parameter `url` is missing", http.StatusBadRequest)
		return
	}

	qualities, cached, err := GetVideoQualities(url)

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot get video qualities", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=14400")

	if cached {
		w.Header().Set("X-Pomu-Cache", "hit")
	} else {
		w.Header().Set("X-Pomu-Cache", "miss")
	}

	if err := json.NewEncoder(w).Encode(qualities); err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot serialize to json", http.StatusInternalServerError)
	}
}
