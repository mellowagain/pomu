package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"golang.org/x/exp/rand"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

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
	rand.Seed(uint64(time.Now().UnixNano()))

	if err := godotenv.Load(); err != nil {
		log.Printf("warning: failed to load .env file: %s\n", err)
	}

	setupSentry()
	checkYouTubeDl()
	checkFfmpeg()
	Scheduler.StartAsync()

	address := os.Getenv("BIND_ADDRESS")

	if len(address) <= 0 {
		address = ":8080"
	}

	db := Connect()
	driver, err := postgres.WithInstance(db, &postgres.Config{})

	if err != nil {
		log.Fatalf("failed to setup database driver: %s", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance("file://migrations", "pomu", driver)

	if err != nil {
		log.Fatalf("failed to initialize migrations: %s", err)
	}

	if err := migrator.Up(); err != nil {
		if err.Error() != "no change" {
			log.Fatalf("failed to run migrations: %s", err)
		}
	}

	app := &Application{
		db:           db,
		secureCookie: setupSecureCookie(),
	}

	go app.restartRecording()

	if strings.ToLower(os.Getenv("HOLODEX_ENABLE")) == "true" {
		log.Println("Holodex auto fetching is enabled")

		if _, err := Scheduler.SingletonMode().Every("1h").StartImmediately().Do(QueueUpcomingStreams, app); err != nil {
			sentry.CaptureException(err)
		}
	}

	setupServer(address, app)
}

func setupServer(address string, app *Application) {
	c := cors.New(cors.Options{
		AllowedOrigins:   strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ","),
		AllowedMethods:   []string{http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	r := mux.NewRouter().StrictSlash(true)

	// == FRONTEND ==
	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./dist/index.html")
	}

	r.HandleFunc("/", serveIndex).Methods("GET")
	r.HandleFunc("/queue", serveIndex).Methods("GET")
	r.HandleFunc("/history", serveIndex).Methods("GET")

	// Static resources
	fileServer := http.FileServer(http.Dir("./dist/assets"))
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fileServer))

	// == API ==

	r.HandleFunc("/api", apiOverview).Methods("GET")
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, "healthy"); err != nil {
			http.Error(w, "unhealthy", http.StatusInternalServerError)
		}
	}).Methods("GET")

	// Videos
	r.HandleFunc("/api/qualities", PeekForQualities).Methods("GET")
	r.HandleFunc("/api/submit", app.SubmitVideo).Methods("POST")
	r.HandleFunc("/api/queue", app.GetQueue).Methods("GET")
	r.HandleFunc("/api/history", app.GetHistory).Methods("GET")

	// Metrics
	r.HandleFunc("/api/logz", app.Log).Methods("GET")
	r.HandleFunc("/api/stats", app.GetStats).Methods("GET")

	// OAuth
	r.HandleFunc("/api/user", app.identifySelf).Methods(http.MethodGet)

	// Discord OAuth
	r.HandleFunc("/oauth/discord", app.DiscordOAuthInitiator).Methods("GET")
	r.HandleFunc("/oauth/discord/redirect", app.DiscordOAuthRedirect).Methods("GET")

	log.Fatal(http.ListenAndServe(address, c.Handler(r)))
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
		log.Fatalf("Failed to find youtube-dl (%s): %s\n", err, output)
	}

	log.Printf("Found youtube-dl version %s\n", strings.TrimSpace(output.String()))
}

func checkFfmpeg() {
	output := new(strings.Builder)
	cmd := exec.Command(os.Getenv("FFMPEG"), "-version")
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to find ffmpeg: %s\n", err)
	}

	before, _, _ := strings.Cut(output.String(), "\n")
	log.Println("Found", before)
}

func (app *Application) restartRecording() {
	queue, err := app.getQueue()

	if err != nil {
		log.Println("Failed to get queue")
		sentry.CaptureException(err)
		return
	}

	log.Println("Found", len(queue), "videos to restart")

	for _, video := range queue {
		videoMetadata, err := GetVideoMetadata(video.Id)
		if err != nil {
			sentry.CaptureException(err)
			log.Println("restart:", video.Id, "Unable to get video metadata")
			continue
		}

		app.scheduleVideo(videoMetadata, video.Id, VideoRequest{
			VideoUrl: fmt.Sprintf("https://youtu.be/%s", video.Id),
			// Use 0 to auto-pick best quality
			Quality: 0,
		})
	}
}

// GitHash will be filled by the build script
var GitHash string

func apiOverview(w http.ResponseWriter, _ *http.Request) {
	if len(GitHash) <= 0 {
		http.Error(w, "pomu was incorrectly built. please see readme.md", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"app":           "pomu.app",
		"documentation": "https://docs.pomu.app",
		"repository":    "https://github.com/mellowagain/pomu",
		"commit":        GitHash,
	}

	SerializeJson(w, response)
}

func (app *Application) identifySelf(w http.ResponseWriter, r *http.Request) {
	user, err := app.ResolveUserFromRequest(r)

	if err != nil || user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	SerializeJson(w, user)
}
