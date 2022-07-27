package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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
	span := sentry.StartSpan(
		context.Background(),
		"youtube-dl get playlist",
		sentry.TransactionName(
			fmt.Sprintf("youtube-dl get playlist %s", p.request.VideoUrl)))

	defer span.Finish()

	output := new(strings.Builder)

	cmd := exec.Command(os.Getenv("YOUTUBE_DL"), "-f", string(strconv.Itoa(int(p.request.Quality))), "-g", p.request.VideoUrl)
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		sentry.CaptureException(err)
		log.Printf("cannot run youtube-dl: %s (output was %s)\n", err, output)
		return "", err
	}
	if strings.Contains(output.String(), "ERROR: This live event will begin in") {
		return "", ErrorLivestreamNotStarted
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

func recordFinished(db *sql.DB, id string, size int64) error {
	tx, err := db.Begin()

	if err != nil {
		return err
	}

	_, err = tx.Exec("update videos set finished = true, file_size = $1 where id = $2", size, id)

	if err != nil {
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
		log.Println("Failed to start ffmpeg:", err)
		hlsClient.Stop()
		return 0, errors.New("failed to start ffmpeg")
	}
	finished := make(chan struct{})

	s3, err := s3.New(os.Getenv("S3_BUCKET"))
	if err != nil {
		sentry.CaptureException(err)
		return 0, errors.New("failed to contact s3 bucket")
	}

	sizeWritten := make(chan int64)

	go func() {
		muxerSpan := span.StartChild("muxer-uploader loop")
		defer muxerSpan.Finish()
		reader, writer := io.Pipe()

		go func() {
			defer func() { finished <- struct{}{} }()
			err := s3.Upload(fmt.Sprintf("%s.mp4", id), reader)
			if err != nil {
				log.Println("s3.Upload2():", err)
				sentry.CaptureException(err)
				return
			} else {
				log.Println("s3 Upload successfully finished")
			}
		}()

		log.Println("Begin copying")
		sentry.CaptureMessage("copy from muxer to s3")
		size, err := io.Copy(writer, muxer)
		if err != nil {
			log.Println("copy muxer to s3:", err)
			sentry.CaptureException(err)
		}
		log.Println("Finished reading from ffmpeg: ", size)
		sizeWritten <- size
		_ = writer.CloseWithError(io.EOF)
	}()

	go func() {
		downloaderSpan := span.StartChild("downloader")
		defer downloaderSpan.Finish()
		video.Download(hlsClient.Segments, muxer)
	}()

	<-finished
	log.Println("record finished")
	return <-sizeWritten, nil
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
				log.Println("Failed record finish for", id)
			}
			return
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
	parsedUrl, err := url.Parse(ytUrl)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if log, ok := ffmpegLogs[parsedUrl.Query().Get("v")]; ok {
		_, err := w.Write([]byte(log.String()))
		if err != nil {
			http.Error(w, "failed to write output bytes", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
