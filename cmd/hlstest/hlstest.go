package main

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"pomu/hls"
	"pomu/qualities"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
)

type ytdlRemotePlaylist struct {
	VideoUrl string
	Quality  int32
}

// ErrorLivestreamNotStarted indicates that the livestream has not started
var ErrorLivestreamNotStarted = errors.New("livestream has not started")

func (p *ytdlRemotePlaylist) Get() (string, error) {
	log.Println("Getting playlist url for", p.VideoUrl)

	span := sentry.StartSpan(
		context.Background(),
		"youtube-dl get playlist",
		sentry.TransactionName(
			fmt.Sprintf("youtube-dl get playlist %s", p.VideoUrl)))
	defer span.Finish()

	// Check that we are trying to record a valid quality
	if p.Quality <= 0 {
		// Stream was queued ahead of time, select best quality
		qualities, _, err := qualities.GetVideoQualities(p.VideoUrl, true)
		if err != nil {
			log.Panicln("Whilst trying to get playlist url, was unable to get qualities for video")
			return "", err
		}
		for _, quality := range qualities {
			if quality.Best {
				log.Println("Whilst trying to get playlist url, automatically chose quality", quality.Code)
				p.Quality = quality.Code
			}
		}
	}

	output := new(strings.Builder)

	cmd := exec.Command(os.Getenv("YOUTUBE_DL"), "-f", string(strconv.Itoa(int(p.Quality))), "-g", p.VideoUrl)
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

	log.Printf("Got url %s\n", stringOutput)

	return stringOutput, nil
}

var _ hls.RemotePlaylist = (*ytdlRemotePlaylist)(nil)

func main() {
	url := os.Args[1]
	quality, err := strconv.Atoi(os.Args[2])

	if err != nil {
		fmt.Println("Code should be int not", os.Args[2])
		return
	}

	downloader := hls.New()
	fmt.Println("Getting segments from", url, "with quality", quality)
	go downloader.Playlist(&ytdlRemotePlaylist{VideoUrl: url, Quality: int32(quality)})

	for segment := range downloader.Segments {
		fmt.Println("Got new segment", segment.Time, segment.Url)
	}
}
