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
	client              *http.Client
	cache               *lru.Cache
	Segments            chan Segment
	playlistUrl         string
	playlistUrlDeadline time.Time
	done                bool
	lastSeq             int
	noChange            int
}

type RemotePlaylist interface {
	// Get gets the playlist url
	Get() (string, error)
}

func (client *Client) getPlaylistUrl(force bool, remotePlaylist RemotePlaylist) (playlistUrl string, err error) {
	if !force && time.Until(client.playlistUrlDeadline) > 0 {
		return client.playlistUrl, nil
	}

	log.Println("Getting playlist url")

	playlistUrl, err = remotePlaylist.Get()
	if err != nil {
		log.Println("Failed to get playlist url:", err)
		return
	}

	client.playlistUrl = playlistUrl
	client.playlistUrlDeadline = time.Now().Add(10 * time.Minute)

	return
}

func (client *Client) getPlaylist(remotePlaylist RemotePlaylist) (m3u8.Playlist, error) {
	playlistUrl, err := client.getPlaylistUrl(false, remotePlaylist)
	if err != nil {
		log.Println("Failed to get playlist url")
		return nil, err
	}

	tries := 0
	var playlist m3u8.Playlist
	for tries >= 0 {
		playlist, err = func() (playlist m3u8.Playlist, err error) {
			request, err := http.NewRequest("GET", playlistUrl, nil)
			if err != nil {
				return nil, err
			}

			request.Header.Set("User-Agent", os.Getenv("HTTP_USERAGENT"))

			resp, err := client.client.Do(request)
			if err != nil {
				// We weren't able to get this url for whatever reason.
				// Try and refresh playlist url and try again
				log.Println("try=", tries, " failed to request playlist:", err)
				tries += 1
				return
			}
			defer resp.Body.Close()

			playlist, _, err = m3u8.DecodeFrom(resp.Body, true)
			if err != nil {
				log.Println("Failed to decode playlist:", err)
			}
			return
		}()

		if err == nil {
			break
		}

		if tries > 20 {
			log.Println("Giving up retrying playlist", err)
			break
		}

		// We failed, re-get playlist url and then try again
		playlistUrl, err = client.getPlaylistUrl(true, remotePlaylist)
		if err != nil {
			log.Println("Failed to get playlist url")
			return nil, err
		}
	}

	return playlist, nil
}

func (client *Client) playlistFrame(start time.Time, remotePlaylist RemotePlaylist) (sleepDuration time.Duration, err error) {
	playlistVariant, err := client.getPlaylist(remotePlaylist)

	if err != nil {
		return 0, err
	}

	switch playlist := playlistVariant.(type) {
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
		log.Println("Unexpected playlist type ", playlistVariant, ", cannot download")
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

		if client.noChange > 40 {
			log.Println("No change in 40 frames, playlist assumed done")
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
		// Allow for a slight buffer of segments
		make(chan Segment, 10),
		"",
		time.Now().Add(-20 * time.Minute),
		false,
		0, 0,
	}
}
