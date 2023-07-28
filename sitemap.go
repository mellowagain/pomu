package main

import (
	"fmt"
	"github.com/ikeikeikeike/go-sitemap-generator/stm"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
)

func (app *Application) Sitemap(w http.ResponseWriter, _ *http.Request) {
	sm, err := app.generateSitemap()

	if err != nil {
		http.Error(w, "failed to generate sitemap", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(sm.XMLContent()); err != nil {
		http.Error(w, "failed to write sitemap", http.StatusInternalServerError)
		log.WithFields(log.Fields{"error": err}).Error("failed to write sitemap")
	}
}

func (app *Application) generateSitemap() (*stm.Sitemap, error) {
	sm := stm.NewSitemap()
	sm.SetDefaultHost(os.Getenv("BASE_URL"))

	sm.Create()

	sm.Add(stm.URL{
		"loc":        "/",
		"changefreq": "hourly",
		"priority":   "0.5",
	})

	sm.Add(stm.URL{
		"loc":        "/queue",
		"changefreq": "hourly",
		"priority":   "0.5",
	})

	sm.Add(stm.URL{
		"loc":        "/history",
		"changefreq": "hourly",
		"priority":   "0.5",
	})

	if err := app.addVideosToSitemap(sm); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to add videos to sitemap")
		return nil, err
	}

	return sm, nil
}

func (app *Application) addVideosToSitemap(sm *stm.Sitemap) error {
	videos, err := AllVideos(app.db)

	if err != nil {
		return err
	}

	for _, video := range videos {
		// thumbnail url
		sm.Add(stm.URL{
			"loc":        fmt.Sprintf("/api/download/%s/thumbnail", video.Id),
			"changefreq": "yearly",
			"priority":   "0.2",
		})

		// video and ffmpeg log is only available after the video is finished
		if !video.Finished {
			continue
		}

		// video download url
		sm.Add(stm.URL{
			"loc":        fmt.Sprintf("/archive/%s", video.Id),
			"changefreq": "yearly",
			"priority":   "0.3",
		})

		// ffmpeg url
		sm.Add(stm.URL{
			"loc":        fmt.Sprintf("/api/download/%s/ffmpeg", video.Id),
			"changefreq": "yearly",
			"priority":   "0.1",
		})
	}

	return nil
}
