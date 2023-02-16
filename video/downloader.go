package video

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"pomu/hls"

	log "github.com/sirupsen/logrus"

	"github.com/getsentry/sentry-go"
)

func logVideo(id string, err error) (entry *log.Entry) {
	if err == nil {
		entry = log.WithFields(log.Fields{"video_id": id})
	} else {
		entry = log.WithFields(log.Fields{"video_id": id, "error": err})
	}
	return
}

// Download downloads segments from the segments channel
// and writes the data into writer w
func Download(id string, segments chan hls.Segment, w io.WriteCloser) {
	defer w.Close()
	client := http.DefaultClient
	failedSegments := 0

	span := sentry.StartSpan(
		context.Background(),
		"Download",
		sentry.TransactionName(
			fmt.Sprintf("Download %s", id)))
	defer span.Finish()

	for segment := range segments {
		req, err := http.NewRequest("GET", segment.Url, nil)
		if err != nil {
			log.Println("http.NewRequest():", err)
			sentry.CaptureException(err)
			return
		}

		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Data: map[string]interface{}{
				"segmentURL":  segment.Url,
				"segmentTIme": segment.Time,
			},
			Level: sentry.LevelInfo,
		})

		req.Header.Set("User-Agent", os.Getenv("HTTP_USERAGENT"))

		resp, err := client.Do(req)
		if err != nil {
			logVideo(id, err).Error("Download failed request for segment", segment.Time, segment.Url)
			sentry.CaptureException(err)
			break
		}

		if !func(resp *http.Response) bool {
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				sentry.AddBreadcrumb(&sentry.Breadcrumb{
					Data: map[string]interface{}{
						"status":           resp.Status,
						"response headers": resp.Header,
					},
					Level: sentry.LevelInfo,
				})
				logVideo(id, nil).Error("Download failed to get segment because of status", resp.StatusCode)
				failedSegments += 1
				return false
			}

			n, err := io.Copy(w, resp.Body)
			if err != nil {
				logVideo(id, err).Error("Download failed to copy segment to writer")
				failedSegments += 1
				sentry.AddBreadcrumb(&sentry.Breadcrumb{
					Message: "Failed to copy segment to writer",
					Level:   sentry.LevelError,
				})
			}
			if resp.ContentLength > n {
				log.Println("video.Download(): io.Copy did not copy enough", n, "copied vs", resp.ContentLength)
				failedSegments += 1
				sentry.AddBreadcrumb(&sentry.Breadcrumb{
					Message: "io.Copy did not copy enough",
					Level:   sentry.LevelError,
				})
			}
			return true
		}(resp) {
			logVideo(id, nil).Error("Failed to download a segment")
		}
	}

	if failedSegments > 0 {
		sentry.CaptureMessage(fmt.Sprint("Download failed ", failedSegments, " segments"))
	}

	logVideo(id, nil).Info("video.Download(): done")
}
