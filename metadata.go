package main

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func GetVideoMetadata(videoId string, token *oauth2.Token) (*youtube.Video, error) {
	service, err := youtube.NewService(context.Background(), option.WithTokenSource(oauth2.StaticTokenSource(token)))

	if err != nil {
		return nil, err
	}

	videoService := youtube.NewVideosService(service)

	list, err := videoService.List([]string{"contentDetails", "liveStreamingDetails"}).Id(videoId).Do()

	if err != nil {
		return nil, err
	}

	if length := len(list.Items); length != 1 {
		return nil, errors.New(fmt.Sprintf("didn't get items, length was %d", length))
	}

	return list.Items[0], nil
}

func IsLivestream(video *youtube.Video) bool {
	return video.LiveStreamingDetails != nil
}

// IsLivestreamStarted checks if the livestream started but has not yet ended
func IsLivestreamStarted(video *youtube.Video) bool {
	return IsLivestream(video) &&
		!IsLivestreamEnded(video) &&
		len(video.LiveStreamingDetails.ScheduledStartTime) > 0 &&
		len(video.LiveStreamingDetails.ActualStartTime) > 0
}

// IsLivestreamEnded checks if the livestream ended
func IsLivestreamEnded(video *youtube.Video) bool {
	return IsLivestream(video) &&
		IsLivestreamStarted(video) &&
		len(video.LiveStreamingDetails.ActualEndTime) > 0
}
