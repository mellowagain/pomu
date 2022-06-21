package main

import (
	"fmt"
	"os"
	"pomu/hls"
)

func main() {
	url := os.Args[1]
	downloader := hls.New()
	fmt.Println("Getting segments from", url)
	go downloader.Playlist(url)

	for segment := range downloader.Segments {
		fmt.Println("Got new segment", segment.Time, segment.Url)
	}
}
