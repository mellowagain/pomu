package main

import (
	"github.com/getsentry/sentry-go"
	"net/http"
	"os"
	"strconv"
)

func (app *Application) GetStats(w http.ResponseWriter, r *http.Request) {
	tx, err := app.db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot start transaction", http.StatusInternalServerError)
		return
	}

	defer tx.Rollback()

	videoAmount := 0
	totalFileSize := 0
	totalLength := 0
	uniqueChannels := 0

	err = tx.QueryRow("select count(id) as video_amount, "+
		"sum(file_size) as total_file_size, "+
		"sum(video_length) as total_length, "+
		"count(distinct(channel_id)) as unique_channels "+
		"from videos").Scan(&videoAmount, &totalFileSize, &totalLength, &uniqueChannels)

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to query for videos", http.StatusInternalServerError)
		return
	}

	usdPerGbPerMonth, err := strconv.ParseFloat(os.Getenv("S3_USD_PER_GB_PER_MONTH"), 64)

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to parse S3_USD_PER_GB_PER_MONTH", http.StatusInternalServerError)
		return
	}

	// 1 gb = 1'000'000'000 bytes
	bytesToGb := float64(totalFileSize) / 1_000_000_000.0
	s3Bill := bytesToGb * usdPerGbPerMonth

	if err := tx.Commit(); err != nil {
		sentry.CaptureException(err)
		http.Error(w, "cannot commit transaction", http.StatusInternalServerError)
		return
	}

	stats := Stats{
		VideoAmount:      videoAmount,
		TotalFileSize:    totalFileSize,
		TotalLength:      totalLength,
		UniqueChannels:   uniqueChannels,
		S3BillPerMonth:   s3Bill,
		UsdPerGbPerMonth: usdPerGbPerMonth,
	}

	SerializeJson(w, stats)
}

type Stats struct {
	VideoAmount      int     `json:"videoAmount"`
	TotalFileSize    int     `json:"totalFileSize"`
	TotalLength      int     `json:"totalLength"`
	UniqueChannels   int     `json:"uniqueChannels"`
	S3BillPerMonth   float64 `json:"s3BillPerMonth"`
	UsdPerGbPerMonth float64 `json:"usdPerGbPerMonth"`
}
