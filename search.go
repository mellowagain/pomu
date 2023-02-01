package main

import (
	"database/sql"
	"encoding/json"
	"github.com/lib/pq"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

const IndexName = "pomu"

func (app *Application) SetupSearch() {
	if strings.ToLower(os.Getenv("MEILISEARCH_ENABLED")) != "true" {
		log.Info("Meilisearch integration is disabled")
		return
	} else {
		log.Info("Meilisearch integration is enabled")
	}

	app.searchClient = meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   os.Getenv("MEILISEARCH_URL"),
		APIKey: os.Getenv("MEILISEARCH_BACKEND_API_KEY"),
	})

	if _, err := app.searchClient.GetIndex(IndexName); err != nil {
		// index does not exist, create it
		_, err := app.searchClient.CreateIndex(&meilisearch.IndexConfig{
			Uid:        IndexName,
			PrimaryKey: "id",
		})

		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("failed to create meilisearch index")
			return
		}
	}

	app.search = app.searchClient.Index(IndexName)

	// upload existing documents to the search instance
	videos, err := allVideos(app.db)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to retrieve newest version of archived videos for the search engine")
		return
	}

	bytes, err := json.Marshal(videos)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to marshal videos into bytes")
		return
	}

	var flattenedVideos []map[string]any

	if err := json.Unmarshal(bytes, &flattenedVideos); err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to unmarshal bytes into videos")
		return
	}

	info, err := app.search.AddDocuments(&flattenedVideos)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to upload newest version of archived videos to search engine")
		return
	}

	distinctTaskInfo, _ := app.search.UpdateDistinctAttribute("id")

	filterableTaskInfo, _ := app.search.UpdateFilterableAttributes(&[]string{
		"id",
		"submitters",
		"scheduledStart",
		"finished",
		"title",
		"channelName",
		"channelId",
		"fileSizeBytes",
		"length",
	})

	sortableTaskInfo, _ := app.search.UpdateSortableAttributes(&[]string{
		"scheduledStart",
		"length",
		"fileSizeBytes",
	})

	log.WithFields(log.Fields{
		"data_task_uid":              info.TaskUID,
		"update_distinct_task_uid":   distinctTaskInfo.TaskUID,
		"update_filterable_task_uid": filterableTaskInfo.TaskUID,
		"update_sortable_task_uid":   sortableTaskInfo.TaskUID,
	}).Info("successfully updated pomu search index with previously archived versions")
}

func SearchMetadata(w http.ResponseWriter, _ *http.Request) {
	enabled := strings.ToLower(os.Getenv("MEILISEARCH_ENABLED")) == "true"

	response := map[string]any{
		"enabled": enabled,
	}

	if enabled {
		response["url"] = os.Getenv("MEILISEARCH_URL")
		response["apiKey"] = os.Getenv("MEILISEARCH_FRONTEND_API_KEY")
	}

	SerializeJson(w, response)
}

func (app *Application) UpsertVideo(video Video) error {
	videos := []Video{video}

	if _, err := app.search.AddDocuments(videos); err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to upsert video")
		return err
	}

	return nil
}

func (video *Video) asMeilisearch() (map[string]any, error) {
	bytes, err := json.Marshal(video)

	if err != nil {
		return nil, err
	}

	var structured map[string]any

	if err := json.Unmarshal(bytes, &structured); err != nil {
		return nil, err
	}

	// meilisearch wants unix timestamp instead of rfc 3339
	structured["start"] = video.Start.Unix()

	return structured, nil
}

func allVideos(db *sql.DB) ([]Video, error) {
	tx, err := db.Begin()

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to start transaction")
		return nil, err
	}

	defer tx.Rollback()

	rows, err := tx.Query("select * from videos order by start")

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to prepare query")
		return nil, err
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Warn("failed to close row")
		}
	}(rows)

	var videos []Video

	for rows.Next() {
		var video Video

		if err := rows.Scan(
			&video.Id,
			pq.Array(&video.Submitters),
			&video.Start,
			&video.Finished,
			&video.Title,
			&video.ChannelName,
			&video.ChannelId,
			&video.Thumbnail,
			&video.FileSize,
			&video.Length); err != nil {
			log.WithFields(log.Fields{"error": err}).Warn("failed to scan row into Video")
			continue
		}

		videos = append(videos, video)
	}

	if err = tx.Commit(); err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to commit transaction")
		return videos, err
	}

	return videos, nil
}
