package hls

import (
	"log"
	"net/http"
	"net/url"
	"strings"
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
	playlistUrl, err := url.Parse(urlString)
	if err != nil {
		return 0, err
	}

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
			var segmentUri string

			if v == nil {
				continue
			}

			if strings.HasPrefix(v.URI, "http") {
				segmentUri, err = url.QueryUnescape(v.URI)
				if err != nil {
					return 0, err
				}
			} else {
				segmentUrl, err := playlistUrl.Parse(v.URI)
				if err != nil {
					return 0, err
				}
				segmentUri, err = url.QueryUnescape(segmentUrl.String())
				if err != nil {
					return 0, err
				}
			}

			// Check whether we have already downloaded this segment
			if _, hit := dl.cache.Get(segmentUri); !hit {
				dl.cache.Add(segmentUri, nil)
				dl.Segments <- &Segment{
					segmentUri,
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
