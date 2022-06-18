package hls

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang/groupcache/lru"
	"github.com/kz26/m3u8"
)

type Segment struct {
	Url  string
	Time time.Duration
}

type Client struct {
	client   *http.Client
	cache    *lru.Cache
	Segments chan *Segment
}

func (dl *Client) playlistFrame(start time.Time, urlString string) (sleepDuration time.Duration, err error) {
	request, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return 0, err
	}

	request.Header.Set("User-Agent", os.Getenv("HTTP_USERAGENT"))

	resp, err := dl.client.Do(request)
	if err != nil {
		return 0, err
	}
	playlist, _, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return 0, err
	}
	// Done with request body
	resp.Body.Close()

	switch playlist := playlist.(type) {
	case *m3u8.MediaPlaylist:
		for _, v := range playlist.Segments {
			if v == nil {
				continue
			}

			// Check whether we have already downloaded this segment
			if _, hit := dl.cache.Get(v.URI); !hit {
				dl.cache.Add(v.URI, nil)
				dl.Segments <- &Segment{
					v.URI,
					time.Since(start),
				}
			}
		}

		if playlist.Closed {
			close(dl.Segments)
			return 0, nil
		}

		return time.Duration(int64(playlist.TargetDuration * float64(time.Second))), nil
	default:
		log.Fatalln("Unexpected playlist type")
	}
	return 0, nil
}

func (dl *Client) Playlist(urlString string) {
	start := time.Now()

	for {
		t, err := dl.playlistFrame(start, urlString)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("Completed frame, resting for ", t, "...")
		time.Sleep(t)
	}

}

func NewDownloader() *Client {
	client := http.DefaultClient
	cache := lru.New(1000)

	return &Client{
		client,
		cache,
		make(chan *Segment),
	}
}
