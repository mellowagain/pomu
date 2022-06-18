package main

import (
	"encoding/json"
	"golang.org/x/oauth2"
	"log"
	"net/http"
)

type VideoRequest struct {
	VideoUrl string `json:"videoUrl"`
	Quality  string `json:"quality"`
}

func (app *Application) SubmitVideo(w http.ResponseWriter, r *http.Request) {
	var request VideoRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "failed to decode json request body", http.StatusInternalServerError)
		return
	}

	cookie, err := r.Cookie("pomu")

	if err != nil {
		http.Error(w, "please login first", http.StatusUnauthorized)
		return
	}

	var token *oauth2.Token

	if err = app.secureCookie.Decode("oauthToken", cookie.Value, token); err != nil {
		http.Error(w, "please login again", http.StatusUnauthorized)
		return
	}

	log.Printf("New video submitted: %s (quality %s)\n", request.VideoUrl, request.Quality)

}

func PeekForQualities(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")

	if len(url) <= 0 {
		http.Error(w, "required parameter `url` is missing", http.StatusBadRequest)
		return
	}

	qualities, cached, err := GetVideoQualities(url)

	if err != nil {
		http.Error(w, "cannot get video qualities", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=14400")

	if cached {
		w.Header().Set("X-Pomu-Cache", "hit")
	} else {
		w.Header().Set("X-Pomu-Cache", "miss")
	}

	if err := json.NewEncoder(w).Encode(qualities); err != nil {
		http.Error(w, "cannot serialize to json", http.StatusInternalServerError)
	}
}
