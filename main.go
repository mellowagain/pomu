package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"os"
)

type Application struct {
	db           *sql.DB
	secureCookie *securecookie.SecureCookie
}

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalln("Failed to load .env file")
	}

	db := Connect()

	address := os.Getenv("BIND_ADDRESS")

	if len(address) <= 0 {
		address = ":8080"
	}

	setupServer(address, &Application{
		db:           db,
		secureCookie: setupSecureCookie(),
	})
}

func setupServer(address string, app *Application) {
	r := mux.NewRouter()

	// Videos
	r.HandleFunc("/qualities", PeekForQualities).Methods("GET")
	r.HandleFunc("/submit", app.SubmitVideo).Methods("POST")

	// OAuth
	r.HandleFunc("/login", OauthLoginHandler).Methods("GET")
	r.HandleFunc("/oauth/redirect", app.OauthRedirectHandler).Methods("GET")

	log.Fatal(http.ListenAndServe(address, r))
}

func setupSecureCookie() *securecookie.SecureCookie {
	hashKey, hashKeyErr := hex.DecodeString(os.Getenv("COOKIE_HASH_KEY"))
	blockKey, blockKeyErr := hex.DecodeString(os.Getenv("COOKIE_BLOCK_KEY"))

	if hashKeyErr != nil || blockKeyErr != nil {
		log.Fatalf("Failed to decode hash key (%s) or block key (%s)\n", hashKeyErr, blockKeyErr)
	}

	if len(hashKey) < 32 || len(blockKey) < 16 {
		log.Printf("Hash key (%d) is less than 32 bytes or block key (%d) is less than 16 bytes. Regenerating\n", len(hashKey), len(blockKey))

		hashKey = securecookie.GenerateRandomKey(32)
		blockKey = securecookie.GenerateRandomKey(16)

		envMap, err := godotenv.Read()

		if err != nil {
			log.Fatalf("Failed to read .env file: %s\n", err)
		}

		envMap["COOKIE_HASH_KEY"] = fmt.Sprintf("%x", hashKey)
		envMap["COOKIE_BLOCK_KEY"] = fmt.Sprintf("%x", blockKey)

		if err = godotenv.Write(envMap, ".env"); err != nil {
			log.Fatalf("Failed to write .env file: %s\n", err)
		}
	}

	return securecookie.New(hashKey, blockKey)
}
