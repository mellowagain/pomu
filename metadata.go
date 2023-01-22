package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func GetVideoMetadataWithToken(videoId string) (*youtube.Video, error) {
	service, err := youtube.NewService(context.Background(), option.WithAPIKey(os.Getenv("GOOGLE_API_KEY")))

	if err != nil {
		return nil, err
	}

	videoService := youtube.NewVideosService(service)

	list, err := videoService.List([]string{"contentDetails", "liveStreamingDetails", "snippet"}).Id(videoId).Do()

	if err != nil {
		return nil, err
	}

	if length := len(list.Items); length != 1 {
		return nil, fmt.Errorf("didn't get items, length was %d", length)
	}

	return list.Items[0], nil
}

func GetVideoMetadata(videoId string) (*youtube.Video, error) {
	service, err := youtube.NewService(
		context.Background(),
		option.WithAPIKey(os.Getenv("GOOGLE_API_KEY")))

	if err != nil {
		return nil, err
	}

	videoService := youtube.NewVideosService(service)

	list, err := videoService.List([]string{"contentDetails", "liveStreamingDetails", "snippet"}).Id(videoId).Do()

	if err != nil {
		return nil, err
	}

	if length := len(list.Items); length != 1 {
		return nil, fmt.Errorf("didn't get items, length was %d", length)
	}

	return list.Items[0], nil
}

func IsLivestream(video *youtube.Video) bool {
	return video != nil && video.LiveStreamingDetails != nil && video.Snippet.LiveBroadcastContent != "none"
}

// IsLivestreamStarted checks if the livestream is currently live
func IsLivestreamStarted(video *youtube.Video) bool {
	return IsLivestream(video) && video.Snippet.LiveBroadcastContent == "live"
}

// IsLivestreamEnded checks if the livestream has ended
func IsLivestreamEnded(video *youtube.Video) bool {
	return IsLivestream(video) && video.Snippet.LiveBroadcastContent == "completed"
}
