package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/patrickmn/go-cache"
)

var qualitiesCache = cache.New(4*time.Hour, 10*time.Minute)

type VideoQuality struct {
	Code       int32  `json:"code"`
	Resolution string `json:"resolution"`
	Best       bool   `json:"best"`
}

func GetVideoQualities(url string, ignoreCache bool) ([]VideoQuality, bool, error) {
	if !isValidUrl(url) {
		return nil, false, errors.New("invalid url")
	}

	videoID := ParseVideoID(url)
	quality, exists := qualitiesCache.Get(videoID)

	if quality != nil && exists && !ignoreCache {
		return quality.([]VideoQuality), true, nil
	}

	span := sentry.StartSpan(context.Background(), "youtube-dl list-formats", sentry.TransactionName(fmt.Sprintf("youtube-dl list-formats %s", url)))

	output := new(strings.Builder)

	cmd := exec.Command(os.Getenv("YOUTUBE_DL"), "--list-formats", url)
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		if strings.Contains(output.String(), "ERROR: This live event will begin in") ||
			strings.Contains(output.String(), "ERROR: Premieres in") {
			return []VideoQuality{{
				Code:       -1,
				Resolution: "Not yet started, will use best quality",
				Best:       false,
			}}, false, nil
		} else {
			sentry.AddBreadcrumb(&sentry.Breadcrumb{Level: sentry.LevelDebug, Message: fmt.Sprintf("ffmpeg output was %s", output)})
			sentry.CaptureException(err)

			log.Printf("failed to run youtube-dl: %s (output was %s)\n", err, output)
			return nil, false, err
		}
	}

	span.Finish()

	split := strings.Split(output.String(), "\n")
	started := false

	var qualities []VideoQuality

	for _, line := range split {
		line = strings.TrimSpace(line)

		if len(line) <= 0 || strings.HasPrefix(line, "[") {
			continue
		}

		if strings.HasPrefix(line, "format code") {
			started = true
			continue
		}

		if !started {
			continue
		}

		var code int32
		var extension string // Unused
		var resolution string

		// format code  extension  resolution note
		// 91           mp4        256x144     269k , avc1.4d400c, 30.0fps, mp4a.40.5
		// [0]			[1]			[2]			[3]

		if _, err := fmt.Sscanf(line, "%d %s %s", &code, &extension, &resolution); err != nil {
			sentry.CaptureException(err)
			continue
		}

		if resolution == "audio" {
			continue
		}

		qualities = append(qualities, VideoQuality{
			Code:       code,
			Resolution: resolution,
			Best:       strings.Contains(line, "(best)"),
		})
	}

	if len(qualities) <= 0 {
		return nil, false, errors.New("unable to find video qualities")
	}

	qualitiesCache.Set(videoID, qualities, 0)

	return qualities, false, nil
}

func isValidUrl(videoUrl string) bool {
	parsedUrl, err := url.Parse(videoUrl)
	loweredHost := strings.ToLower(parsedUrl.Host)

	hasYouTube := strings.Contains(loweredHost, "youtube.com")
	hasYouTuDotBe := strings.Contains(loweredHost, "youtu.be")

	return err == nil && len(parsedUrl.Scheme) > 0 && (hasYouTube || hasYouTuDotBe)
}

func ParseVideoID(videoUrl string) string {
	parsedUrl, _ := url.Parse(videoUrl)

	switch strings.TrimPrefix(parsedUrl.Host, "www.") {
	case "youtu.be":
		// https://youtu.be/2naTB5J0jfI
		return parsedUrl.Path[1:]
	case "youtube.com":
		// https://www.youtube.com/watch?v=2naTB5J0jfI
		return parsedUrl.Query().Get("v")
	}

	sentry.CaptureMessage("Failed to parse video url \"" + videoUrl + "\" into video id")
	return videoUrl
}
