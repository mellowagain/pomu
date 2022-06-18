package main

import (
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Failed to load .env file")
	}

	Connect()

	address := os.Getenv("BIND_ADDRESS")

	if len(address) <= 0 {
		address = ":8080"
	}

	setupServer(address)
}

func setupServer(address string) {
	log.Fatal(http.ListenAndServe(address, nil))
}
