package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	sentrylogrus "github.com/getsentry/sentry-go/logrus"
	"github.com/meilisearch/meilisearch-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/rand"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"

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

	searchClient *meilisearch.Client
	search       *meilisearch.Index
}

func main() {
	initLogging()
	rand.Seed(uint64(time.Now().UnixNano()))

	if err := godotenv.Load(); err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("failed to load .env file")
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
		log.WithFields(log.Fields{"error": err}).Fatal("failed to setup connect to database")
	}

	migrator, err := migrate.NewWithDatabaseInstance("file://migrations", "pomu", driver)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("failed to initialize migrations")
	}

	if err := migrator.Up(); err != nil {
		if err.Error() != "no change" {
			log.WithFields(log.Fields{"error": err}).Fatal("failed to run migrations")
		}
	}

	app := &Application{
		db:           db,
		secureCookie: setupSecureCookie(),
	}

	go app.restartRecording()
	go app.SetupSearch()

	if strings.ToLower(os.Getenv("HOLODEX_ENABLE")) == "true" {
		log.Info("holodex auto fetching is enabled")

		if _, err := Scheduler.SingletonMode().Every("1h").StartImmediately().Do(QueueUpcomingStreams, app); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("failed to schedule task for queuing upcoming holodex streams")
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

	// Prometheus middleware
	middleware := NewPrometheusMiddleware(prometheus.DefaultRegisterer, nil)

	// == FRONTEND ==
	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./dist/index.html")
	}

	r.HandleFunc("/", middleware.WrapHandler("/", http.HandlerFunc(serveIndex))).Methods("GET")
	r.HandleFunc("/queue", middleware.WrapHandler("/queue", http.HandlerFunc(serveIndex))).Methods("GET")
	r.HandleFunc("/history", middleware.WrapHandler("/history", http.HandlerFunc(serveIndex))).Methods("GET")

	// Static resources
	fileServer := http.FileServer(http.Dir("./dist/assets"))
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fileServer))

	// Prometheus
	r.Handle("/metrics", middleware.WrapHandler("/metrics", promhttp.Handler()))

	// Sitemap
	r.Handle("/sitemap.xml", middleware.WrapHandler("/sitemap.xml", http.HandlerFunc(app.Sitemap))).Methods("GET")

	// == API ==

	r.HandleFunc("/api", middleware.WrapHandler("/api", http.HandlerFunc(apiOverview))).Methods("GET")
	r.HandleFunc("/healthz", middleware.WrapHandler("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := fmt.Fprintln(w, "healthy"); err != nil {
			http.Error(w, "unhealthy", http.StatusInternalServerError)
		}
	}))).Methods("GET")

	// Videos
	r.HandleFunc("/api/validate", middleware.WrapHandler("/api/validate", http.HandlerFunc(app.ValidateLivestream))).Methods("GET")
	r.HandleFunc("/api/qualities", middleware.WrapHandler("/api/qualities", http.HandlerFunc(PeekForQualities))).Methods("GET")
	r.HandleFunc("/api/submit", middleware.WrapHandler("/api/submit", http.HandlerFunc(app.SubmitVideo))).Methods("POST")
	r.HandleFunc("/api/queue", middleware.WrapHandler("/api/queue", http.HandlerFunc(app.GetQueue))).Methods("GET")
	r.HandleFunc("/api/history", middleware.WrapHandler("/api/history", http.HandlerFunc(app.GetHistory))).Methods("GET")
	r.HandleFunc("/api/search", middleware.WrapHandler("/api/search", http.HandlerFunc(SearchMetadata))).Methods("GET")

	// Specific video
	r.HandleFunc("/api/video/{id}/downloads", middleware.WrapHandler("/api/video/{id}/downloads", http.HandlerFunc(app.DownloadCount))).Methods("GET")

	// Downloads
	// TODO: move this into the /api/video group, smth like /api/video/{id}/download/{type}
	r.HandleFunc("/api/download/{id}/{type}", middleware.WrapHandler("/api/download/{id}/{type}", http.HandlerFunc(app.VideoDownload))).Methods("GET", "HEAD")

	// Metrics
	r.HandleFunc("/api/logz", middleware.WrapHandler("/api/logz", http.HandlerFunc(app.Log))).Methods("GET", "HEAD")
	r.HandleFunc("/api/stats", middleware.WrapHandler("/api/stats", http.HandlerFunc(app.GetStats))).Methods("GET")

	// Users
	r.HandleFunc("/logout", middleware.WrapHandler("/logout", http.HandlerFunc(app.Logout))).Methods("POST")
	r.HandleFunc("/api/user", middleware.WrapHandler("/api/user", http.HandlerFunc(app.IdentitySelf))).Methods("GET")
	r.HandleFunc("/api/user/{provider}/{id}", middleware.WrapHandler("/api/user/{provider}/{id}", http.HandlerFunc(app.Identity))).Methods("GET")

	// Discord OAuth
	r.HandleFunc("/oauth/discord", middleware.WrapHandler("/oauth/discord", http.HandlerFunc(app.DiscordOAuthInitiator))).Methods("GET")
	r.HandleFunc("/oauth/discord/redirect", middleware.WrapHandler("/oauth/discord/redirect", http.HandlerFunc(app.DiscordOAuthRedirect))).Methods("GET")

	log.Fatal(http.ListenAndServe(address, c.Handler(r)))
}

func setupSecureCookie() *securecookie.SecureCookie {
	hashKey, hashKeyErr := hex.DecodeString(os.Getenv("COOKIE_HASH_KEY"))
	blockKey, blockKeyErr := hex.DecodeString(os.Getenv("COOKIE_BLOCK_KEY"))

	if hashKeyErr != nil || blockKeyErr != nil {
		log.WithFields(log.Fields{
			"hash_key_error":  hashKeyErr,
			"block_key_error": blockKeyErr,
		}).Fatal("failed to decode hash key or block key")
	}

	if len(hashKey) < 32 || len(blockKey) < 16 {
		log.WithFields(log.Fields{
			"hash_key_length":  len(hashKey),
			"block_key_length": len(blockKey),
		}).Info("hash key is less than 32 bytes or block key is less than 16 bytes. regenerating.")

		hashKey = securecookie.GenerateRandomKey(32)
		blockKey = securecookie.GenerateRandomKey(16)

		envMap, err := godotenv.Read()

		if err != nil {
			log.WithFields(log.Fields{"error": err}).Warn("failed to load .env file")
		}

		envMap["COOKIE_HASH_KEY"] = fmt.Sprintf("%x", hashKey)
		envMap["COOKIE_BLOCK_KEY"] = fmt.Sprintf("%x", blockKey)

		if err = godotenv.Write(envMap, ".env"); err != nil {
			log.WithFields(log.Fields{"error": err}).Fatal("failed to write .env file")
		}
	}

	return securecookie.New(hashKey, blockKey)
}

func setupSentry() {
	if strings.ToLower(os.Getenv("SENTRY_ENABLE")) != "true" {
		log.Info("sentry error reporting is disabled")
		return
	} else {
		log.Info("sentry error reporting is enabled")
	}

	sampleRate, err := strconv.ParseFloat(os.Getenv("SENTRY_SAMPLE_RATE"), 64)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("failed to parse `SENTRY_SAMPLE_RATE` environment variable")
	}

	err = sentry.Init(sentry.ClientOptions{
		AttachStacktrace: true,
		Debug:            strings.ToLower(os.Getenv("SENTRY_DEBUG")) == "true",
		TracesSampleRate: sampleRate,
	})

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("failed to setup sentry")
	}

	levels := []log.Level{log.ErrorLevel, log.FatalLevel, log.PanicLevel}
	hook := sentrylogrus.NewFromClient(levels, sentry.CurrentHub().Client())

	defer hook.Flush(5 * time.Second)
	log.AddHook(hook)

	log.RegisterExitHandler(func() {
		// if log.Fatal gets called, exit(1) will be executed which means no `defer`s (defined above) run, so flush manually
		hook.Flush(5 * time.Second)
	})
}

func checkYouTubeDl() {
	output := new(strings.Builder)

	cmd := exec.Command(os.Getenv("YT_DLP"), "--version")
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"output": output,
		}).Fatal("failed to find youtube-dl")
	}

	log.WithFields(log.Fields{"version": strings.TrimSpace(output.String())}).Info("found youtube-dl")
}

func checkFfmpeg() {
	output := new(strings.Builder)
	cmd := exec.Command(os.Getenv("FFMPEG"), "-version")
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Run(); err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("failed to find ffmpeg")
	}

	firstLine, _, _ := strings.Cut(output.String(), "\n")
	nonTrimmedVersion, _, _ := strings.Cut(firstLine, " Copyright (c) 2000-")
	version := nonTrimmedVersion[15:]

	log.WithFields(log.Fields{"version": version}).Info("found ffmpeg")
}

func initLogging() {
	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation:    true,
		PadLevelText:              true,
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		QuoteEmptyFields:          true,
	})
}

func (app *Application) restartRecording() {
	queue, err := app.getQueue()

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to get queue from database")
		return
	}

	log.WithFields(log.Fields{"amount": len(queue)}).Info("found pending videos to restart")

	for _, video := range queue {
		videoMetadata, err := GetVideoMetadata(video.Id)

		if err != nil {
			log.WithFields(log.Fields{
				"video_id": video.Id,
				"error":    err,
			}).Error("unable to get video meta data from youtube")
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
