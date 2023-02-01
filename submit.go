package main

import (
	"database/sql"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
	"pomu/qualities"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/lib/pq"
	"golang.org/x/exp/slices"
	"google.golang.org/api/youtube/v3"
)

type Video struct {
	Id          string    `json:"id"`
	Submitters  []string  `json:"submitters"`
	Start       time.Time `json:"scheduledStart"`
	Finished    bool      `json:"finished"`
	Title       string    `json:"title"`
	ChannelName string    `json:"channelName"`
	ChannelId   string    `json:"channelId"`
	Thumbnail   string    `json:"thumbnail"`
	DownloadUrl string    `json:"downloadUrl,omitempty"` // Not actually part of the query
	FileSize    string    `json:"fileSizeBytes,omitempty"`
	Length      string    `json:"length,omitempty"`
}

type VideoRequest struct {
	VideoUrl string `json:"videoUrl"`
	Quality  int32  `json:"quality"`
}

func (r *VideoRequest) Id() (string, error) {
	return qualities.ParseVideoID(r.VideoUrl), nil
}

func (app *Application) SubmitVideo(w http.ResponseWriter, r *http.Request) {
	var request VideoRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "failed to decode json request body", http.StatusInternalServerError)
		return
	}

	user, err := app.ResolveUserFromRequest(r)

	if user == nil || err != nil {
		http.Error(w, "please login first", http.StatusUnauthorized)
		return
	}

	videoId := qualities.ParseVideoID(request.VideoUrl)

	videoMetadata, err := GetVideoMetadataWithToken(videoId)

	if err != nil {
		http.Error(w, "failed to get video metadata", http.StatusBadRequest)
		return
	}

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

	valid, err := CheckChannelAgainstHolodex(videoMetadata.Snippet.ChannelId)

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to check channel against holodex", http.StatusInternalServerError)
		return
	}

	if !valid {
		http.Error(w, "only livestreams by holodex listed vtubers are allowed", http.StatusBadRequest)
		return
	}

	startTime, err := GetVideoStartTime(videoMetadata)

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

	defer tx.Rollback()

	var video Video
	err = tx.QueryRow("select * from videos where id = $1 limit 1", videoId).Scan(
		&video.Id,
		pq.Array(&video.Submitters),
		&video.Start,
		&video.Finished,
		&video.Title,
		&video.ChannelName,
		&video.ChannelId,
		&video.Thumbnail,
		&video.FileSize,
		&video.Length)

	var reschedule bool

	if err != nil {
		if err != sql.ErrNoRows {
			sentry.CaptureException(err)
			http.Error(w, "failed to check if video already is being archived", http.StatusInternalServerError)
			return
		}

		thumbnailUrl, err := SaveThumbnail(videoId, FindSuitableThumbnail(videoMetadata.Snippet.Thumbnails))

		if err != nil {
			http.Error(w, "Failed to save thumbnail for video "+videoId, http.StatusInternalServerError)
			return
		}

		statement, err := tx.Prepare("insert into videos (id, submitters, start, title, channel_name, channel_id, thumbnail) values ($1, $2, $3, $4, $5, $6, $7) returning *")

		if err != nil {
			sentry.CaptureException(err)
			http.Error(w, "failed to prepare statement", http.StatusInternalServerError)
			return
		}

		row := statement.QueryRow(videoId,
			pq.Array([]string{user.Provider + "/" + user.Id}),
			startTime,
			videoMetadata.Snippet.Title,
			videoMetadata.Snippet.ChannelTitle,
			videoMetadata.Snippet.ChannelId,
			thumbnailUrl)

		if err := row.Err(); err != nil {
			sentry.CaptureException(err)
			http.Error(w, "failed to create new video", http.StatusInternalServerError)
			return
		}

		if err = row.Scan(&video.Id,
			pq.Array(&video.Submitters),
			&video.Start,
			&video.Finished,
			&video.Title,
			&video.ChannelName,
			&video.ChannelId,
			&video.Thumbnail,
			&video.FileSize,
			&video.Length); err != nil {
			sentry.CaptureException(err)
			http.Error(w, "failed to create new video", http.StatusInternalServerError)
			return
		}

		reschedule = true
	} else {
		if !slices.Contains(video.Submitters, user.Provider+"/"+user.Id) {
			statement, err := tx.Prepare("update videos set submitters = array_append(submitters, $1), start = $2 where id = $3 returning *")

			if err != nil {
				sentry.CaptureException(err)
				http.Error(w, "failed to prepare statement", http.StatusInternalServerError)
				return
			}

			if err := statement.QueryRow(user.Provider+"/"+user.Id, startTime, video.Id).
				Scan(&video.Id,
					pq.Array(&video.Submitters),
					&video.Start,
					&video.Finished,
					&video.Title,
					&video.ChannelName,
					&video.ChannelId,
					&video.Thumbnail,
					&video.FileSize,
					&video.Length); err != nil {
				sentry.CaptureException(err)
				log.Println(err)
				http.Error(w, "failed to update existing video", http.StatusInternalServerError)
				return
			}
		}

		reschedule = false
	}

	log.Printf("New video submitted: %s (quality %d)\n", request.VideoUrl, request.Quality)

	if reschedule {
		err := app.scheduleVideo(videoMetadata, videoId, request)
		if err != nil {
			http.Error(w, "Failed to schedule video recording", http.StatusInternalServerError)
			return
		}

		go app.UpsertVideo(video)
	}

	if err := tx.Commit(); err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Expires", strings.ReplaceAll(startTime.UTC().Format(time.RFC1123), "UTC", "GMT"))

	SerializeJson(w, video)
}

func GetVideoStartTime(videoMetadata *youtube.Video) (startTime time.Time, err error) {
	if IsLivestreamStarted(videoMetadata) {
		startTime, err = time.Parse(time.RFC3339, videoMetadata.LiveStreamingDetails.ActualStartTime)
	} else {
		startTime, err = time.Parse(time.RFC3339, videoMetadata.LiveStreamingDetails.ScheduledStartTime)
	}
	return startTime, err
}

func (app *Application) scheduleVideo(
	videoMetadata *youtube.Video,
	videoId string,
	request VideoRequest) error {

	if IsLivestreamStarted(videoMetadata) {
		if _, err := Scheduler.SingletonMode().Every("10s").LimitRunsTo(1).Tag(videoId).StartImmediately().Do(StartRecording, app, request); err != nil {
			sentry.CaptureException(err)
			return err
		}

		log.Printf("Livestream already started, starting recording immediatly")
	} else {
		startTime, err := GetVideoStartTime(videoMetadata)
		if err != nil {
			return err
		}
		if _, err := Scheduler.SingletonMode().Every("10s").LimitRunsTo(1).StartAt(startTime).Tag(videoId).Do(StartRecording, app, request); err != nil {
			sentry.CaptureException(err)
			return err
		}

		log.Printf("Livestream recording scheduled for %s", startTime.Format(time.RFC3339))
	}

	return nil
}

func PeekForQualities(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")

	if len(url) <= 0 {
		http.Error(w, "required parameter `url` is missing", http.StatusBadRequest)
		return
	}

	qualities, cached, err := qualities.GetVideoQualities(url, false)

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot get video qualities", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "max-age=14400")

	if cached {
		w.Header().Set("X-Pomu-Cache", "hit")
	} else {
		w.Header().Set("X-Pomu-Cache", "miss")
	}

	SerializeJson(w, qualities)
}
