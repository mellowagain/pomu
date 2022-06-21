package video

import (
	"io"
	"log"
	"net/http"
	"os"
	"pomu/hls"
)

// Download downloads segments from the segments channel
// and writes the data into writer w
func Download(segments chan hls.Segment, w io.WriteCloser) {
	defer w.Close()
	client := http.DefaultClient
	for segment := range segments {
		req, err := http.NewRequest("GET", segment.Url, nil)
		if err != nil {
			log.Fatalln("http.NewRequest():", err)
		}

		req.Header.Set("User-Agent", os.Getenv("HTTP_USERAGENT"))

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			log.Println("client.Get():", resp.StatusCode, err)
			// If we didn't get a 200 then we are probably done
			break
		}

		func(body io.ReadCloser) {
			defer body.Close()

			n, err := io.Copy(w, body)
			if err != nil {
				log.Println("video.Download(): io.Copy():", err)
			}
			if resp.ContentLength > n {
				log.Println("video.Download(): io.Copy did not copy enough", n, "copied vs", resp.ContentLength)
			}
		}(resp.Body)
	}
	log.Println("video.Download(): done")
}
