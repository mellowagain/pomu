package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"pomu/hls"
	"pomu/qualities"
	"pomu/s3"
	"pomu/video"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"

	"github.com/getsentry/sentry-go"
)

type ytdlRemotePlaylist struct {
	request VideoRequest
}

// ErrorLivestreamNotStarted indicates that the livestream has not started
var ErrorLivestreamNotStarted = errors.New("livestream has not started")

func (p *ytdlRemotePlaylist) Get() (string, error) {
	log.Println("Getting playlist url for", p.request.VideoUrl)

	span := sentry.StartSpan(
		context.Background(),
		"youtube-dl get playlist",
		sentry.TransactionName(
			fmt.Sprintf("youtube-dl get playlist %s", p.request.VideoUrl)))
	defer span.Finish()

	// Check that we are trying to record a valid quality
	if p.request.Quality <= 0 {
		// Stream was queued ahead of time, select best quality
		qualities, _, err := qualities.GetVideoQualities(p.request.VideoUrl, true)
		if err != nil {
			log.Panicln("Whilst trying to get playlist url, was unable to get qualities for video")
			return "", err
		}
		for _, quality := range qualities {
			if quality.Best {
				log.Println("Whilst trying to get playlist url, automatically chose quality", quality.Code)
				p.request.Quality = quality.Code
			}
		}
	}

	output := new(strings.Builder)

	cmd := exec.Command(os.Getenv("YOUTUBE_DL"), "-f", string(strconv.Itoa(int(p.request.Quality))), "-g", p.request.VideoUrl)
	cmd.Stdout = output
	cmd.Stderr = output

	err := cmd.Run()

	// NOTE(emily): If the livestream has not started yet, ytdl will return 1
	// We want to check first whether the live event WILL begin, and return the correct
	// error
	if strings.Contains(output.String(), "ERROR: This live event will begin in") {
		return "", ErrorLivestreamNotStarted
	}

	if err != nil {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{Level: sentry.LevelDebug, Message: fmt.Sprintf("ffmpeg output was %s", output)})
		sentry.CaptureException(err)
		log.Printf("cannot run youtube-dl: %s (output was %s)\n", err, output)
		return "", err
	}

	span.Finish()
	stringOutput := strings.TrimSpace(output.String())

	if !strings.HasSuffix(stringOutput, ".m3u8") {
		log.Printf("Expected m3u8 output, received %s\n", stringOutput)
		return "", errors.New("expected m3u8")
	}

	return stringOutput, nil
}

var _ hls.RemotePlaylist = (*ytdlRemotePlaylist)(nil)

var ffmpegLogs = make(map[string]*strings.Builder)

func hasLivestreamStarted(request VideoRequest) (bool, error) {
	_, err := (&ytdlRemotePlaylist{request}).Get()
	if err == ErrorLivestreamNotStarted {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func videoLengthFromLog(id string) time.Duration {
	// NOTE(emily): Here we can get the video length by looking at the ffmpeg log
	logString := strings.TrimSpace(ffmpegLogs[id].String())
	timeStart := strings.LastIndex(logString, "time=") + len("time=")
	timeEnd := strings.Index(logString[timeStart:], " ")

	timeStr := logString[timeStart : timeStart+timeEnd]
	// replace punctuation with their respective time units
	timeStr = strings.Replace(timeStr, ":", "h", 1)
	timeStr = strings.Replace(timeStr, ":", "m", 1)
	timeStr = strings.Replace(timeStr, ".", "s", 1)
	timeStr = timeStr + "ms"

	duration, err := time.ParseDuration(timeStr)
	if err != nil {
		log.Println("Failed to parse duration from ffmpeg log")
	}

	return duration
}

func (app *Application) recordFinished(db *sql.DB, id string, size int64) error {
	tx, err := db.Begin()

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to begin transaction")
		return err
	}

	defer tx.Rollback()

	length := videoLengthFromLog(id)

	log.WithFields(log.Fields{
		"id":     id,
		"size":   size,
		"length": length,
	}).Info("finishing video")

	var video Video

	statement, err := tx.Prepare("update videos set finished = true, file_size = $1, video_length = $2 where id = $3 returning *")

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to prepare statement")
		return err
	}

	row := statement.QueryRow(size, int(length.Seconds()), id)

	if err := row.Err(); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to execute statement")
		return err
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
		log.WithFields(log.Fields{"error": err}).Error("failed to serialize row into video")
		return err
	}

	if err := tx.Commit(); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to commit transaction")
		return err
	}

	go app.UpsertVideo(video)
	return nil
}

func recordFailed(db *sql.DB, id string) error {
	log.Println("Record for", id, "failed, deleting from database")

	// TODO(emily): Probably want to make sure that s3 is cleaned up as well

	tx, err := db.Begin()

	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.Exec("delete from videos where id = $1", id)

	if err != nil {
		return err
	}

	if tx.Commit() != nil {
		return err
	}

	return nil
}

func record(request VideoRequest) (size int64, err error) {
	log.Println("Starting recording of ", request.VideoUrl)
	span := sentry.StartSpan(
		context.Background(),
		"record",
		sentry.TransactionName(
			fmt.Sprintf("record %s", request.VideoUrl)))
	defer span.Finish()

	id, err := request.Id()
	if err != nil {
		log.Println("failed to get video id: ", err)
		sentry.CaptureException(err)
		return
	}
	// Start getting segments
	hlsClient := hls.New(id)
	defer hlsClient.Stop()
	go func() {
		hlsClientPlaylistSpan := span.StartChild("hls-client playlist")
		defer hlsClientPlaylistSpan.Finish()
		logVideo(request, nil).Info("Starting HLS Client")
		defer logVideo(request, nil).Info("HLS Client stopped")
		hlsClient.Playlist(&ytdlRemotePlaylist{request})
	}()

	// Start the video muxer
	muxer := &video.Muxer{}
	ffmpegLogs[id] = new(strings.Builder)
	muxer.Stderr = ffmpegLogs[id]
	err = muxer.Start()
	if err != nil {
		log.Println(id, "Failed to start ffmpeg:", err)
		sentry.CaptureException(err)
		return 0, errors.New("failed to start ffmpeg")
	}
	finished := make(chan struct{})
	defer close(finished)

	s3, err := s3.New(os.Getenv("S3_BUCKET"))
	if err != nil {
		sentry.CaptureException(err)
		return 0, errors.New("failed to contact s3 bucket")
	}

	sizeWritten := make(chan int64)
	defer close(sizeWritten)

	go func() {
		muxerSpan := span.StartChild("muxer-uploader loop")
		defer muxerSpan.Finish()
		reader, writer := io.Pipe()

		go func() {
			defer func() { finished <- struct{}{} }()
			err := s3.Upload(fmt.Sprintf("%s.mp4", id), reader, "video/mp4")
			if err != nil {
				log.Println(id, "s3.Upload2():", err)
				sentry.CaptureException(err)
				return
			}

			log.Println(id, "s3 Upload successfully finished")
		}()

		logVideo(request, nil).Info("Begin copying")
		// sentry.AddBreadcrumb(&sentry.Breadcrumb{Message: "copy from muxer to s3"})
		size, err := io.Copy(writer, muxer)
		if err != nil {
			logVideo(request, err).Error("copy muxer to s3:", err)
			sentry.CaptureException(err)
		}
		logVideo(request, nil).Info(id, "Finished reading from ffmpeg: ", size)
		// NOTE(emily): Must close first before writing.
		// sizeWritten <- size will block until its read
		// but it wont be read until s3 finishes, which is after the writer
		// has closed.
		_ = writer.CloseWithError(io.EOF)
		sizeWritten <- size
	}()

	go func() {
		downloaderSpan := span.StartChild("downloader")
		defer downloaderSpan.Finish()
		logVideo(request, nil).Info("Starting segment downloader")
		defer logVideo(request, nil).Info("Segment downloader stopped")
		video.Download(id, hlsClient.Segments, muxer)
	}()

	<-finished
	log.Println(id, "record finished")
	go uploadLog(s3, id)
	return <-sizeWritten, nil
}

func uploadLog(s3 *s3.Client, id string) {
	ffmpegLog := ffmpegLogs[id].String()
	lines := strings.Split(ffmpegLog, "\n")

	if len(lines) > 3 {
		lines = lines[3:]
	}

	err := s3.Upload(fmt.Sprintf("%s.log", id), strings.NewReader(strings.Join(lines, "\n")), "text/plain")
	if err != nil {
		log.Println(id, "uploadLog: s3.Upload2():", err)
		sentry.CaptureException(err)
		return
	}
}

func logVideo(request VideoRequest, err error) (entry *log.Entry) {
	if err == nil {
		entry = log.WithFields(log.Fields{"video_url": request.VideoUrl})
	} else {
		entry = log.WithFields(log.Fields{"video_url": request.VideoUrl, "error": err})
	}
	return
}

func StartRecording(app *Application, request VideoRequest) {
	logVideo(request, nil).Info("Start recording")
	id, err := request.Id()
	if err != nil {
		logVideo(request, err).Error("Failed to get video id")
		return
	}
	// See if this video has been re-scheduled into the future...
	metadata, err := GetVideoMetadata(id)

	if err != nil {
		logVideo(request, err).Error("Failed to get metadata for scheduled video")
		return
	}

	newStartTime, err := GetVideoStartTime(metadata)
	if err != nil {
		logVideo(request, err).Error("Failed to parse new start time from metadata for video")
		return
	}

	const RETRY_INTERVAL = 1 * time.Minute
	const MAX_RETRIES = 120
	const MAX_DURATION = RETRY_INTERVAL * MAX_RETRIES

	if time.Until(newStartTime) > (RETRY_INTERVAL * MAX_RETRIES) {
		logVideo(request, nil).Info("video has been moved to more than", MAX_DURATION.String(), "into the future, rescheduling")
		// Schedule a new cronjob that will re-queue the video
		if _, err := Scheduler.
			SingletonMode().
			LimitRunsTo(1).
			StartAt(time.Now().Add(RETRY_INTERVAL)).
			Tag("Reschedule"+request.VideoUrl).
			Do(app.scheduleVideo, metadata, id, metadata); err != nil {
			logVideo(request, err).Error("Failed to reschedule video")
		}
		return
	}

	for try := 0; try < MAX_RETRIES; try += 1 {
		if started, err := hasLivestreamStarted(request); err == nil && started {
			size, err := record(request)
			if err != nil {
				log.Println("record failed:", err)
				return
			}
			err = app.recordFinished(app.db, id, size)
			if err != nil {
				logVideo(request, err).Error("Failed record finish")
			}
			return
		} else if err == ErrorLivestreamNotStarted {
			logVideo(request, nil).Info("Livestream has not started yet")
		} else if err != nil {
			logVideo(request, err).Error("Failed checking livestream started")
			err = recordFailed(app.db, id)
			if err != nil {
				logVideo(request, err).Error("Failed recordFailed")
			}
			return
		}

		logVideo(request, nil).Info("Waiting for video, try=", try)
		time.Sleep(RETRY_INTERVAL)
	}
}

func (app *Application) Log(w http.ResponseWriter, r *http.Request) {
	ytUrl := r.URL.Query().Get("url")
	var id string
	if len(ytUrl) == 0 {
		id = r.URL.Query().Get("id")
	} else {
		id = qualities.ParseVideoID(ytUrl)
	}
	if log, ok := ffmpegLogs[id]; ok {
		_, err := w.Write([]byte(log.String()))
		if err != nil {
			http.Error(w, "failed to write output bytes", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
