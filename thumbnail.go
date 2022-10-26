package main

import (
	"fmt"
	"google.golang.org/api/youtube/v3"
	"log"
	"net/http"
	"os"
	"pomu/s3"
)

// FindSuitableThumbnail returns the highest resolution thumbnail url which is available
func FindSuitableThumbnail(details *youtube.ThumbnailDetails) string {
	qualities := []*youtube.Thumbnail{
		details.Maxres,
		details.High,
		details.Medium,
		details.Standard,
		details.Default,
	}

	for _, thumbnail := range qualities {
		if thumbnail != nil {
			return thumbnail.Url
		}
	}

	return ""
}

// SaveThumbnail saves a thumbnail to S3
func SaveThumbnail(id string, url string) (string, error) {
	s3Client, err := s3.New(os.Getenv("S3_BUCKET"))

	if err != nil {
		log.Printf("[warn] failed to create s3 client in order to upload thumbnail: %s\n", err)
		return url, err
	}

	response, err := http.Get(url)

	if err != nil {
		log.Printf("[warn] failed to get thumbnail from youtube: %s\n", err)
		return url, err
	}

	if err := s3Client.Upload(fmt.Sprintf("%s.jpg", id), response.Body); err != nil {
		log.Println("s3 thumbnail upload failed:", err)
		return url, err
	}

	return fmt.Sprintf("%s/%s.jpg", os.Getenv("S3_DOWNLOAD_URL"), id), nil
}
