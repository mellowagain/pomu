package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (app *Application) GetHistory(w http.ResponseWriter, r *http.Request) {
	page, limit, err := parsePageAndLimit(r.URL.Query())
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

	whereClause := ""

	if !showUnfinished {
		whereClause = "where finished = true"
	}

	rows, err := tx.Query(fmt.Sprintf("select * from videos %s order by start limit %d offset %d", whereClause, limit, page*limit))

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

	var videos []Video

	for rows.Next() {
		var video Video

		if err := rows.Scan(&video.Id, &video.Submitters, &video.Start, &video.Finished); err != nil {
			sentry.CaptureException(err)
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

// parsePageAndLimit returns the page and limit. If not set, will return default values
func parsePageAndLimit(values url.Values) (int, int, error) {
	pageStr := values.Get("page")
	var page int

	if len(pageStr) > 0 {
		convertedPage, err := strconv.Atoi(pageStr)

		if err != nil {
			return 0, 0, err
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
			return 0, 0, err
		}

		limit = min(convertedLimit, 100)
	} else {
		limit = 25
	}

	return page, limit, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
