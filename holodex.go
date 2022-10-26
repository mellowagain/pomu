package main

import (
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func QueueUpcomingStreams(app *Application) {
	orgs := strings.Split(os.Getenv("HOLODEX_ORGS"), ",")

	for _, org := range orgs {
		streams, err := queryUpcomingStreams(org)

		if err != nil {
			log.Printf("failed to query upcoming streams for org %s\n", org)
		}

		log.Printf("found %d streams for %s matching holodex criteria\n", len(streams), org)

		for _, stream := range streams {
			if len(stream.Id) <= 0 {
				log.Printf("%s has no reservation up, skipping\n", stream.Id)
				continue
			}

			tx, err := app.db.Begin()

			if err != nil {
				sentry.CaptureException(err)
				log.Printf("failed to start transaction: %s\n", err)
				continue
			}

			var video Video
			err = tx.QueryRow("select * from videos where id = $1 limit 1", stream.Id).Scan(
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

			// Video already exists in db, skip it
			if err == nil {
				log.Printf("skipping %s as it already is scheduled to be saved", stream.Id)
				continue
			}

			videoMetadata, err := GetVideoMetadata(stream.Id)
			if err != nil {
				sentry.CaptureException(err)
				log.Printf("failed to get video meta data for %s: %s", stream.Id, err)
				continue
			}

			startTime, err := GetVideoStartTime(videoMetadata)

			if err != nil {
				sentry.CaptureException(err)
				log.Printf("failed to get video start time for %s: %s\n", stream.Id, err)
				continue
			}

			thumbnailUrl, err := SaveThumbnail(stream.Id, FindSuitableThumbnail(videoMetadata.Snippet.Thumbnails))

			if err != nil {
				sentry.CaptureException(err)
				log.Printf("failed to save thumbnail for %s: %s\n", stream.Id, err)
				continue
			}

			statement, err := tx.Prepare("insert into videos (id, submitters, start, title, channel_name, channel_id, thumbnail) values ($1, $2, $3, $4, $5, $6, $7) returning *")

			if err != nil {
				sentry.CaptureException(err)
				log.Printf("failed to create video for %s\n", stream.Id)
				continue
			}

			row := statement.QueryRow(stream.Id, pq.Array([]string{"pomu.app"}), startTime, videoMetadata.Snippet.Title, videoMetadata.Snippet.ChannelTitle, videoMetadata.Snippet.ChannelId, thumbnailUrl)

			if err := row.Err(); err != nil {
				sentry.CaptureException(err)
				log.Printf("failed to create video for %s\n", stream.Id)
				continue
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
				log.Printf("failed to get video for %s\n", stream.Id)
				sentry.CaptureException(err)

				continue
			}

			if err := tx.Commit(); err != nil {
				sentry.CaptureException(err)
				log.Printf("failed to commit transaction: %s\n", err)
				continue
			}

			err = app.scheduleVideo(videoMetadata, video.Id, VideoRequest{
				VideoUrl: fmt.Sprintf("https://youtu.be/%s", video.Id),
				// Use 0 to auto-pick best quality
				Quality: 0,
			})

			if err != nil {
				log.Printf("failed to automatically schedule video %s: %s\n", video.Id, err)
				continue
			}

			log.Printf("Automatically scheduled %s (title: \"%s\") for %s\n", video.Id, video.Title, startTime.Format(time.RFC1123))
		}
	}
}

type UpcomingStreamChannel struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Organization string `json:"org"`
	EnglishName  string `json:"english_name"`
}

type UpcomingStream struct {
	Id               string                `json:"id"`
	Title            string                `json:"title"`
	StartedScheduled string                `json:"start_scheduled"`
	Channel          UpcomingStreamChannel `json:"channel"`
}

func queryUpcomingStreams(organization string) ([]UpcomingStream, error) {
	request, err := http.NewRequest("GET", "https://holodex.net/api/v2/live", nil)

	if err != nil {
		fmt.Printf("Failed to start new request to holodex: %s", err)
		return nil, err
	}

	request.Header.Set("X-APIKEY", os.Getenv("HOLODEX_API_KEY"))
	request.Header.Set("User-Agent", "pomu.app")

	query := request.URL.Query()

	query.Set("include", "live_info")
	query.Set("limit", "50")
	query.Set("topic", os.Getenv("HOLODEX_TOPIC"))
	query.Set("type", "stream")
	query.Set("status", "upcoming")
	query.Set("max_upcoming_hours", "24") // We query every hour anyways
	query.Set("org", organization)

	request.URL.RawQuery = query.Encode()

	response, err := http.DefaultClient.Do(request)
	defer response.Body.Close()

	if err != nil {
		fmt.Printf("Failed to send request to holodex: %s", err)
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Printf("Failed to read response from holodex: %s", err)
		return nil, err
	}

	var results []UpcomingStream

	if err := json.Unmarshal(body, &results); err != nil {
		fmt.Printf("Failed to unmarshal json response from holodex: %s", err)
		return nil, err
	}

	return results, nil
}
