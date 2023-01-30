package video

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"pomu/hls"

	"github.com/getsentry/sentry-go"
)

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

		req.Header.Set("User-Agent", os.Getenv("HTTP_USERAGENT"))

		resp, err := client.Do(req)
		if err != nil {
			log.Println("video.Download(): client.Get():", err)
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
				log.Println("video.Download(): client.Get():", resp.StatusCode)
				failedSegments += 1
				return false
			}

			n, err := io.Copy(w, resp.Body)
			if err != nil {
				log.Println("video.Download(): io.Copy():", err)
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
			log.Println("Failed to download a segment, stopping.")
		}
	}

	if failedSegments > 0 {
		sentry.CaptureMessage(fmt.Sprint("Download failed", failedSegments, "segments"))

	}

	log.Println("video.Download(): done")
}
