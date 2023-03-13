package qualities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/patrickmn/go-cache"
)

var qualitiesCache = cache.New(4*time.Hour, 10*time.Minute)

type VideoQuality struct {
	Code       int32   `json:"code"`
	Resolution string  `json:"resolution"`
	Vbr        float64 `json:"-"`
	Best       bool    `json:"best"`
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

	cmd := exec.Command(os.Getenv("YT_DLP"), "-j", "--list-formats", url)
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		if strings.Contains(output.String(), "This live event will begin in") ||
			strings.Contains(output.String(), "Premieres in") {
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

	stdout := output.String()
	jsonBegin := strings.Index(stdout, "{")

	type jsonMap map[string]any
	var v jsonMap

	if err := json.Unmarshal([]byte(stdout[jsonBegin:]), &v); err != nil {
		return nil, false, errors.New("failed to parse yt-dlp output")
	}

	var qualities []VideoQuality

	formats := v["formats"].([]any)

	for _, format := range formats {
		format := format.(map[string]any)

		code, err := strconv.Atoi(format["format_id"].(string))

		if err != nil {
			continue
		}

		qualities = append(qualities, VideoQuality{
			Code:       int32(code),
			Resolution: format["resolution"].(string),
			Vbr:        format["vbr"].(float64),
			Best:       false,
		})
	}

	if len(qualities) <= 0 {
		return nil, false, errors.New("unable to find video qualities")
	}

	highestIndex := 0
	highestVbr := 0.0

	for index, quality := range qualities {
		if highestVbr == 0.0 {
			highestIndex = index
			highestVbr = quality.Vbr
		} else if quality.Vbr > highestVbr {
			highestIndex = index
			highestVbr = quality.Vbr
		}
	}

	qualities[highestIndex].Best = true

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
