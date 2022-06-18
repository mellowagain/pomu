package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"pomu/hls"
	"pomu/uploader"

	pomuFs "pomu/fs"

	"github.com/joho/godotenv"
)

func writeFile(segments chan []byte, body io.ReadCloser) {
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		log.Fatalln("ioutil.ReadAll():", err)
	}

	segments <- data
}

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Failed to load .env file")
	}

	playlistUrl := os.Args[1]
	path := os.Args[2]
	fmt.Println("Getting segments from", playlistUrl, "and uploading to", os.Getenv("S3_BUCKET"))

	downloader := hls.NewDownloader()

	fs := pomuFs.NewFilesystemFS("cmd/uploader/uploaded")

	segmentWriter := uploader.NewSegmentWriter(fs, path)

	segments := make(chan []byte, 100)
	uploader := uploader.New(segments, segmentWriter, 10*1000*1000)
	go downloader.Playlist(playlistUrl)
	go uploader.ProcessSegments()

	client := http.DefaultClient
	for segment := range downloader.Segments {
		fmt.Println("Got new segment", segment.Time)

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

		writeFile(segments, resp.Body)
	}
}
