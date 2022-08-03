package video

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"pomu/hls"

	"github.com/getsentry/sentry-go"
)

// Download downloads segments from the segments channel
// and writes the data into writer w
func Download(segments chan hls.Segment, w io.WriteCloser) {
	defer w.Close()
	client := http.DefaultClient
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

		func(resp *http.Response) {
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				sentry.AddBreadcrumb(&sentry.Breadcrumb{
					Data: map[string]interface{}{
						"status":           resp.Status,
						"response headers": resp.Header,
					},
					Level: sentry.LevelInfo,
				})
				sentry.CaptureMessage(fmt.Sprint("video.Download(): client.Get():", resp.StatusCode))
				log.Println("video.Download(): client.Get():", resp.StatusCode)
				return
			}

			n, err := io.Copy(w, resp.Body)
			if err != nil {
				log.Println("video.Download(): io.Copy():", err)
				sentry.CaptureException(err)
			}
			if resp.ContentLength > n {
				log.Println("video.Download(): io.Copy did not copy enough", n, "copied vs", resp.ContentLength)
			}
		}(resp)
	}
	log.Println("video.Download(): done")
}
