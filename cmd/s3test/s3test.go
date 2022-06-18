package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"pomu/s3"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Failed to load .env file")
	}

	testTxt, _ := os.Open("cmd/s3test/test.txt")
	defer testTxt.Close()

	buffer, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal("ioutil.ReadAll(): ", err)
	}
	data := bytes.NewReader(buffer)

	client, err := s3.New(os.Getenv("S3_BUCKET"))
	if err != nil {
		log.Fatalln("s3.New(): ", err)
	}
	err = client.Upload("test/s3test.txt", data)
	if err != nil {
		log.Fatalln("client.Upload(): ", err)
	}
}
