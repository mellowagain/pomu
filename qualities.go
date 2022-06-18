package main

import (
	"errors"
	"fmt"
	"github.com/patrickmn/go-cache"
	"log"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

var qualitiesCache = cache.New(4*time.Hour, 10*time.Minute)

type VideoQuality struct {
	Code       int32  `json:"code"`
	Resolution string `json:"resolution"`
	Best       bool   `json:"best"`
}

func GetVideoQualities(url string) ([]VideoQuality, bool, error) {
	if !isValidUrl(url) {
		return nil, false, errors.New("invalid url")
	}

	videoID := ParseVideoID(url)
	quality, exists := qualitiesCache.Get(videoID)

	if quality != nil && exists {
		return quality.([]VideoQuality), true, nil
	}

	output := new(strings.Builder)

	cmd := exec.Command("youtube-dl", "--list-formats", url)
	cmd.Stdout = output

	if err := cmd.Run(); err != nil {
		log.Println("cannot run")
		return nil, false, err
	}

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

		if strings.HasPrefix(line, "ERROR: This live event will begin in") {
			qualities = append(qualities, VideoQuality{
				Code:       -1,
				Resolution: "Not yet started, will use best quality",
				Best:       false,
			})

			return qualities, false, nil
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
		return parsedUrl.Path
	case "youtube.com":
		// https://www.youtube.com/watch?v=2naTB5J0jfI
		return parsedUrl.Query().Get("v")
	}

	return videoUrl
}
