package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"

	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
)

// Application is a shared state struct between all web routes
type Application struct {
	db           *sql.DB
	secureCookie *securecookie.SecureCookie
}

func main() {
	log.SetPrefix("[Pomu] ")

	if err := godotenv.Load(); err != nil {
		log.Fatalln("Failed to load .env file")
	}

	setupSentry()
	checkYouTubeDl()
	Scheduler.StartAsync()

	address := os.Getenv("BIND_ADDRESS")

	if len(address) <= 0 {
		address = ":8080"
	}

	setupServer(address, &Application{
		db:           Connect(),
		secureCookie: setupSecureCookie(),
	})
}

func setupServer(address string, app *Application) {
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./dist/index.html")
	}).Methods("GET")

	// Static resources @ /public
	fileServer := http.FileServer(http.Dir("./dist/assets"))
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fileServer))

	// Videos
	r.HandleFunc("/qualities", PeekForQualities).Methods("GET")
	r.HandleFunc("/submit", app.SubmitVideo).Methods("POST")
	r.HandleFunc("/queue", app.GetQueue).Methods("GET")
	r.HandleFunc("/history", app.GetHistory).Methods("GET")

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
			sentry.CaptureException(err)
			log.Fatalf("Failed to read .env file: %s\n", err)
		}

		envMap["COOKIE_HASH_KEY"] = fmt.Sprintf("%x", hashKey)
		envMap["COOKIE_BLOCK_KEY"] = fmt.Sprintf("%x", blockKey)

		if err = godotenv.Write(envMap, ".env"); err != nil {
			sentry.CaptureException(err)
			log.Fatalf("Failed to write .env file: %s\n", err)
		}
	}

	return securecookie.New(hashKey, blockKey)
}

func setupSentry() {
	if strings.ToLower(os.Getenv("SENTRY_ENABLE")) != "true" {
		log.Println("Sentry error reporting is disabled")
		return
	} else {
		log.Println("Sentry error reporting is enabled")
	}

	sampleRate, err := strconv.ParseFloat(os.Getenv("SENTRY_SAMPLE_RATE"), 64)

	if err != nil {
		log.Fatalf("Failed to parse SENTRY_SAMPLE_RATE: %s\n", err)
	}

	err = sentry.Init(sentry.ClientOptions{
		AttachStacktrace: true,
		Debug:            strings.ToLower(os.Getenv("SENTRY_DEBUG")) == "true",
		TracesSampleRate: sampleRate,
	})

	if err != nil {
		log.Fatalf("Failed to setup sentry: %s\n", err)
		return
	}
}

func checkYouTubeDl() {
	output := new(strings.Builder)

	cmd := exec.Command(os.Getenv("YOUTUBE_DL"), "--version")
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to find youtube-dl: %s\n", err)
	}

	log.Printf("Found youtube-dl version %s\n", strings.TrimSpace(output.String()))
}
