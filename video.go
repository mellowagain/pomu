package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"pomu/hls"
	"pomu/s3"
	"pomu/video"
	"strconv"
	"strings"
	"time"

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
		qualities, _, err := GetVideoQualities(p.request.VideoUrl, true)
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

var ffmpegLogs map[string]*strings.Builder = make(map[string]*strings.Builder)

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

func recordFinished(db *sql.DB, id string, size int64) error {
	tx, err := db.Begin()

	if err != nil {
		return err
	}

	length := videoLengthFromLog(id)

	log.Println("Finishing video", id, "with size", size, "and length", length)

	_, err = tx.Exec(
		"update videos set finished = true, file_size = $1, video_length = $2 where id = $3", size, int(length.Seconds()), id)

	if err != nil {
		log.Println("Failed to update video as finished:", err)
		sentry.CaptureException(err)
		return err
	}

	if tx.Commit() != nil {
		return err
	}

	return nil
}

func recordFailed(db *sql.DB, id string) error {
	log.Println("Record for", id, "failed, deleting from database")

	// TODO(emily): Probably want to make sure that s3 is cleaned up as well

	tx, err := db.Begin()
	if err != nil {
		return err
	}
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
	hlsClient := hls.New()
	go func() {
		hlsClientPlaylistSpan := span.StartChild("hls-client playlist")
		defer hlsClientPlaylistSpan.Finish()
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
		hlsClient.Stop()
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
			err := s3.Upload(fmt.Sprintf("%s.mp4", id), reader)
			if err != nil {
				log.Println(id, "s3.Upload2():", err)
				sentry.CaptureException(err)
				return
			}

			log.Println(id, "s3 Upload successfully finished")
		}()

		log.Println(id, "Begin copying")
		sentry.AddBreadcrumb(&sentry.Breadcrumb{Message: "copy from muxer to s3"})
		size, err := io.Copy(writer, muxer)
		if err != nil {
			log.Println("copy muxer to s3:", err)
			sentry.CaptureException(err)
		}
		log.Println(id, "Finished reading from ffmpeg: ", size)
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
		video.Download(id, hlsClient.Segments, muxer)
	}()

	<-finished
	log.Println(id, "record finished")
	go uploadLog(s3, id)
	return <-sizeWritten, nil
}

func uploadLog(s3 *s3.Client, id string) {
	ffmpegLog := ffmpegLogs[id].String()

	err := s3.Upload(fmt.Sprintf("%s.log", id), strings.NewReader(ffmpegLog))
	if err != nil {
		log.Println(id, "uploadLog: s3.Upload2():", err)
		sentry.CaptureException(err)
		return
	}
}

func StartRecording(db *sql.DB, request VideoRequest) {
	log.Println("Waiting for", request.VideoUrl)
	id, err := request.Id()
	if err != nil {
		log.Println("failed to get video id for", request.VideoUrl)
		return
	}
	for try := 0; try < 120; try += 1 {
		if started, err := hasLivestreamStarted(request); err == nil && started {
			size, err := record(request)
			if err != nil {
				log.Println("record failed:", err)
				return
			}

			err = recordFinished(db, id, size)
			if err != nil {
				log.Println("Failed record finish for", id, ":", err)
			}
			return
		} else if err == ErrorLivestreamNotStarted {
			log.Println("Livestream has not started yet")
		} else if err != nil {
			log.Println("Failed checking livestream started:", err)
			err = recordFailed(db, id)
			if err != nil {
				log.Println("Failed record fail for", id)
			}
			return
		}

		log.Println("Waiting for", request.VideoUrl, "try=", try)
		time.Sleep(1 * time.Minute)
	}
}

func (app *Application) Log(w http.ResponseWriter, r *http.Request) {
	ytUrl := r.URL.Query().Get("url")
	var id string
	if len(ytUrl) == 0 {
		id = r.URL.Query().Get("id")
	} else {
		id = ParseVideoID(ytUrl)
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
