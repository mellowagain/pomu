package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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

// func buildYtdlCmd(url string, quality int32) (cmd *exec.Cmd, output *strings.Builder) {
// 	output = new(strings.Builder)
// 	cmd = exec.Command(os.Getenv("YOUTUBE_DL"), "-f", strconv.Itoa(int(quality)), "-g", url)
// 	cmd.Stdout = output
// 	cmd.Stderr = output
// 	return
// }

type ytdlPlaylistUrl struct {
	request VideoRequest
}

func (p *ytdlPlaylistUrl) Get() (string, error) {
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
		return "", errors.New("livestream has not started")
	}

	span.Finish()
	stringOutput := strings.TrimSpace(output.String())

	if !strings.HasSuffix(stringOutput, ".m3u8") {
		log.Printf("Expected m3u8 output, received %s\n", stringOutput)
		return "", errors.New("expcted m3u8")
	}
	return stringOutput, nil
}

var _ hls.RemotePlaylist = (*ytdlPlaylistUrl)(nil)

func hasLivestreamStarted(request VideoRequest) bool {
	if _, err := (&ytdlPlaylistUrl{request}).Get(); err != nil {
		return false
	}
	return true
}

func record(request VideoRequest) {
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
		hlsClient.Playlist(&ytdlPlaylistUrl{request})
	}()

	// Start the video muxer
	muxer := &video.Muxer{}
	stderr := strings.Builder{}
	muxer.Stderr = &stderr
	muxer.Start()

	finished := make(chan struct{})

	s3, err := s3.New(os.Getenv("S3_BUCKET"))
	if err != nil {
		sentry.CaptureException(err)
		return
	}

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
			} else {
				log.Println("s3 Upload successfully finished")
			}
		}()

		log.Println("Begin copying")
		sentry.CaptureMessage("copy from muxer to s3")
		n, err := io.Copy(writer, muxer)
		if err != nil {
			log.Fatalln("io.Copy():", err)
			sentry.CaptureException(err)
		}
		log.Println("Finished reading from ffmpeg: ", n)
		writer.CloseWithError(io.EOF)
	}()

	go func() {
		downloaderSpan := span.StartChild("downloader")
		defer downloaderSpan.Finish()
		video.Download(hlsClient.Segments, muxer)
	}()

	<-finished
}

func StartRecording(request VideoRequest) {
	log.Println("Waiting for ", request.VideoUrl)
	for try := 0; try < 120; try += 1 {
		if hasLivestreamStarted(request) {
			record(request)
			return
		}

		log.Println("Waiting for ", request.VideoUrl, " try=", try)
		time.Sleep(1 * time.Minute)
	}
}
