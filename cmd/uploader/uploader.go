package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"pomu/hls"
	"pomu/s3"
	"pomu/video"

	"github.com/joho/godotenv"
)

type StaticPlaylistUrl struct {
	url string
}

func (p *StaticPlaylistUrl) Get() (string, error) {
	return p.url, nil
}

var _ hls.RemotePlaylist = (*StaticPlaylistUrl)(nil)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Failed to load .env file")
	}

	playlistUrl := os.Args[1]
	path := os.Args[2]
	fmt.Println("Getting segments from", playlistUrl, "and uploading to", path)

	hlsClient := hls.New()

	s3, err := s3.New(os.Getenv("S3_BUCKET"))
	if err != nil {
		log.Fatalln("s3.New():", err)
	}

	go hlsClient.Playlist(&StaticPlaylistUrl{url: playlistUrl})

	muxer := &video.Muxer{}
	muxer.Stderr, err = os.OpenFile(fmt.Sprintf("cmd/uploader/uploaded/%s.ffmpeg", path), os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Fatalln("os.OpenFile():", err)
	}
	err = muxer.Start()
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		reader, writer := io.Pipe()

		go func() {
			err := s3.Upload(fmt.Sprintf("%s.mp4", path), reader)
			if err != nil {
				log.Println("s3.Upload2():", err)
			} else {
				log.Println("s3 Upload successfully finished")
			}

		}()

		log.Println("Begin copying")

		n, err := io.Copy(writer, muxer)
		if err != nil {
			log.Fatalln("io.Copy():", err)
		}
		log.Println("Finished reading from ffmpeg: ", n)
		writer.CloseWithError(io.EOF)
	}()

	go video.Download(hlsClient.Segments, muxer)

	fmt.Println("Press any key to stop...")

	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')

	hlsClient.Stop()

	fmt.Println("Press any key to exit...")
	reader.ReadString('\n')
	os.Exit(0)
}
