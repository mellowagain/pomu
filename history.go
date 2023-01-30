package main

import (
	"database/sql"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/lib/pq"
)

func (app *Application) GetHistory(w http.ResponseWriter, r *http.Request) {
	page, limit, sort, err := parseFilterArgs(r.URL.Query())
	showUnfinished := strings.ToLower(r.URL.Query().Get("unfinished")) == "true"

	if err != nil {
		http.Error(w, "invalid page or limit parameter", http.StatusBadRequest)
		return
	}

	tx, err := app.db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot start transaction", http.StatusInternalServerError)
		return
	}

	defer tx.Rollback()

	whereClause := ""

	if !showUnfinished {
		whereClause = "where finished = true"
	}

	rows, err := tx.Query(fmt.Sprintf("select * from videos %s order by start %s limit %d offset %d", whereClause, sort, limit+1, page*limit))

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to query for videos", http.StatusInternalServerError)
		return
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			sentry.CaptureException(err)
		}
	}(rows)

	videos := []Video{}
	hasMore := false

	for rows.Next() {
		var video Video

		if err := rows.Scan(&video.Id, pq.Array(&video.Submitters), &video.Start, &video.Finished, &video.Title, &video.ChannelName, &video.ChannelId, &video.Thumbnail, &video.FileSize, &video.Length); err != nil {
			sentry.CaptureException(err)
			continue
		}

		if video.Finished {
			video.DownloadUrl = fmt.Sprintf("/api/download/%s/video", video.Id)
		}

		videos = append(videos, video)
	}

	if len(videos) == (limit + 1) {
		hasMore = true
		videos = videos[:len(videos)-1]
	}

	videoCount := 0

	if err := tx.QueryRow(fmt.Sprintf("select count(*) from videos %s", whereClause)).Scan(&videoCount); err != nil {
		log.Printf("%s\n", err)
		sentry.CaptureException(err)
		http.Error(w, "failed to query total video count", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")

	w.Header().Set("X-Pomu-Pagination-Total", strconv.Itoa(videoCount))

	if hasMore {
		w.Header().Set("X-Pomu-Pagination-Has-More", "true")
	} else {
		w.Header().Set("X-Pomu-Pagination-Has-More", "false")
	}

	SerializeJson(w, videos)
}

// parseFilterArgs returns the page, limit and sort. If not set, will return default values
func parseFilterArgs(values url.Values) (int, int, string, error) {
	pageStr := values.Get("page")
	var page int

	if len(pageStr) > 0 {
		convertedPage, err := strconv.Atoi(pageStr)

		if err != nil {
			return 0, 0, "asc", err
		}

		page = convertedPage
	} else {
		page = 0
	}

	limitStr := values.Get("limit")
	var limit int

	if len(limitStr) > 0 {
		convertedLimit, err := strconv.Atoi(limitStr)

		if err != nil {
			return 0, 0, "asc", err
		}

		limit = min(convertedLimit, 100)
	} else {
		limit = 25
	}

	sortStr := values.Get("sort")
	sort := "asc"

	if len(sortStr) > 0 {
		switch strings.ToLower(sortStr) {
		case "asc":
			sort = "asc"
		case "desc":
			sort = "desc"
		default:
			return 0, 0, "asc", errors.New("only asc and desc allowed for sort")
		}
	}

	return page, limit, sort, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
