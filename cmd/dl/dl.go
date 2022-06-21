package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"pomu/hls"

	"github.com/joho/godotenv"
)

func writeFile(path string, body io.ReadCloser, segmentID int) {
	defer body.Close()

	os.MkdirAll("cmd/dl/downloaded", fs.ModePerm)
	file, err := os.OpenFile(fmt.Sprintf("cmd/dl/downloaded/%s.ts", path), os.O_CREATE|os.O_APPEND, fs.ModePerm)
	if err != nil {
		log.Fatalln("os.OpenFile():", err)
	}

	_, err = io.Copy(file, body)
	if err != nil {
		log.Fatalln("io.Copy()", err)
	}
}

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Failed to load .env file")
	}

	playlistUrl := os.Args[1]
	path := os.Args[2]
	downloader := hls.New()
	fmt.Println("Getting segments from", playlistUrl)
	go downloader.Playlist(playlistUrl)

	segmentID := 0
	client := http.DefaultClient
	for segment := range downloader.Segments {
		fmt.Println("Got new segment", segment.Time, segment.Url)

		req, err := http.NewRequest("GET", segment.Url, nil)
		if err != nil {
			log.Fatalln("http.NewRequest():", err)
		}

		req.Header.Set("User-Agent", os.Getenv("HTTP_USERAGENT"))

		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			log.Println("client.Get():", resp.Body, resp.StatusCode, err, resp)
			continue
		}

		writeFile(path, resp.Body, segmentID)

		segmentID += 1
	}
}
