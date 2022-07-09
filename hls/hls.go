package hls

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
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
	Segments chan Segment
	done     bool
	lastSeq  int
	noChange int
}

type RemotePlaylist interface {
	// Get gets the playlist url
	Get() (string, error)
}

func getPlaylist(client *http.Client, remotePlaylist RemotePlaylist) (m3u8.Playlist, error) {
	playlistUrl, err := remotePlaylist.Get()
	if err != nil {
		log.Println("Failed to get playlist url")
		return nil, err
	}

	request, err := http.NewRequest("GET", playlistUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", os.Getenv("HTTP_USERAGENT"))

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	playlist, _, err := m3u8.DecodeFrom(resp.Body, true)
	if err != nil {
		return nil, err
	}
	// Done with request body
	err = resp.Body.Close()
	if err != nil {
		log.Fatalln("resp.Body.Close()", err)
	}

	return playlist, nil
}

func (client *Client) playlistFrame(start time.Time, remotePlaylist RemotePlaylist) (sleepDuration time.Duration, err error) {
	playlist, err := getPlaylist(client.client, remotePlaylist)

	if err != nil {
		return 0, err
	}

	switch playlist := playlist.(type) {
	case *m3u8.MediaPlaylist:
		if playlist.SeqNo == uint64(client.lastSeq) {
			log.Println("Seq did not change", client.noChange)
			client.noChange += 1
		} else {
			client.noChange = 0
		}

		for _, v := range playlist.Segments {
			if client.done {
				return 0, nil
			}

			if v == nil {
				continue
			}

			// Check whether we have already downloaded this segment
			if _, hit := client.cache.Get(v.URI); !hit {
				client.cache.Add(v.URI, nil)
				client.Segments <- Segment{
					v.URI,
					time.Since(start),
				}
			}
		}

		client.lastSeq = int(playlist.SeqNo)

		if playlist.Closed {
			return 0, nil
		}

		return time.Duration(int64(playlist.TargetDuration * float64(time.Second))), nil
	default:
		log.Fatalln("Unexpected playlist type")
	}
	return 0, nil
}

func (client *Client) Playlist(playlist RemotePlaylist) {
	start := time.Now()
	defer close(client.Segments)

	for !client.done {
		t, err := client.playlistFrame(start, playlist)
		if err != nil {
			log.Println("Failed playlist frame:", err)
			sentry.CaptureMessage(fmt.Sprint("failed playlist frame: ", err))
			return
		}

		if t == 0 {
			log.Println("Time to sleep is 0, done")
			return
		}

		if client.noChange > 10 {
			log.Println("No change in 10 frames, playlist assumed done")
			sentry.CaptureMessage("playlist did not change in 10 frame, done")
			return
		}

		// log.Println("Completed frame, resting for ", t, "...")
		time.Sleep(t)
	}

	log.Println("HLS CLient finished")
}

func (client *Client) Stop() {
	client.done = true
}

func New() *Client {
	client := http.DefaultClient
	cache := lru.New(1000)

	return &Client{
		client,
		cache,
		make(chan Segment),
		false,
		0, 0,
	}
}
