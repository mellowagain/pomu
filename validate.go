package main

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"net/http"
	"os"
	"pomu/qualities"
)

func (app *Application) ValidateLivestream(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")

	if len(url) <= 0 {
		http.Error(w, "required parameter `url` is missing", http.StatusBadRequest)
		return
	}

	id := qualities.ParseVideoID(url)

	if id == url {
		http.Error(w, "failed to parse video id from url", http.StatusInternalServerError)
		return
	}

	video, err := GetVideoMetadata(id)

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to get video meta data from youtube api", http.StatusInternalServerError)
		return
	}

	valid, err := CheckChannelAgainstHolodex(video.Snippet.ChannelId)

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to check channel against holodex", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"channelId": video,
		"valid":     valid,
	}

	SerializeJson(w, response)
}

func CheckChannelAgainstHolodex(channelId string) (bool, error) {
	// if the submissions are not restricted to only vtubers, always return true as we don't need to check against holodex then
	if os.Getenv("RESTRICT_VTUBER_SUBMISSIONS") != "true" {
		return true, nil
	}

	request, err := http.NewRequest("GET", fmt.Sprintf("https://holodex.net/api/v2/channels/%s", channelId), nil)

	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("failed to start new request to holodex: %s", err)
		return false, err
	}

	request.Header.Set("X-APIKEY", os.Getenv("HOLODEX_API_KEY"))
	request.Header.Set("User-Agent", "pomu.app")

	response, err := http.DefaultClient.Do(request)
	defer response.Body.Close()

	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("failed to send request to holodex: %s", err)
		return false, err
	}

	// if the search does not result in a 404, they are a valid vtuber or clipper
	return response.StatusCode == http.StatusOK, nil
}
