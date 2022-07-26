package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/getsentry/sentry-go"
	"github.com/hymkor/go-lazy"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googleOauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

var oAuthConfig = lazy.New(func() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Scopes:       []string{googleOauth2.UserinfoProfileScope, youtube.YoutubeUploadScope, youtube.YoutubeScope, youtube.YoutubeReadonlyScope},
	}
})

func OauthLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Google itself does not recommend a `state` (csrf token) so keep it empty
	url := oAuthConfig.Value().AuthCodeURL("")

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (app *Application) OauthRedirectHandler(w http.ResponseWriter, r *http.Request) {
	if oauthErr := r.URL.Query().Get("oauth_err"); len(oauthErr) > 0 {
		http.Redirect(w, r, "/?oauthError="+oauthErr, http.StatusTemporaryRedirect)
		return
	}

	code := r.URL.Query().Get("code")
	scope := r.URL.Query().Get("scope")

	if len(code) <= 0 || len(scope) <= 0 {
		http.Error(w, "code and scope cannot be empty", http.StatusBadRequest)
		return
	}

	token, err := oAuthConfig.Value().Exchange(context.Background(), code)

	if err != nil {
		http.Error(w, "failed to exchange code into token. please retry", http.StatusBadGateway)
		return
	}

	cookie, err := app.secureCookie.Encode("oauthToken", token)

	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "failed to set cookie", http.StatusBadGateway)
		return
	}

	w.Header().Set("Set-Cookie", "pomu="+cookie+"; Path=/; Max-Age=3600")

	service, err := googleOauth2.NewService(context.Background(), option.WithTokenSource(oauth2.StaticTokenSource(token)))

	if err != nil {
		http.Error(w, "failed to create service", http.StatusInternalServerError)
		return
	}

	userInfoService := googleOauth2.NewUserinfoService(service)
	info, err := userInfoService.Get().Do()

	if err != nil {
		http.Error(w, "failed to get user info", http.StatusInternalServerError)
		log.Println("Failed to get user info: ", err)
		return
	}

	user, err := GetUser(info.Id, app.db)

	if err != nil {
		http.Error(w, "failed to get user info", http.StatusInternalServerError)
		log.Println("Failed to get user info: ", err)
		return
	}

	redirectUrl := "/?success"

	if user == nil {
		if user, err = CreateUser(info, app.db); err != nil {
			sentry.CaptureException(err)
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		redirectUrl += "Register"
	}

	http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
}

func (app *Application) ResolveUserFromRequest(r *http.Request) (user *User, err error) {
	cookie, err := r.Cookie("pomu")

	if err != nil {
		return
	}

	var token *oauth2.Token

	if err = app.secureCookie.Decode("oauthToken", cookie.Value, &token); err != nil {
		return
	}

	user, err = ResolveUser(token, app.db)

	return
}

func (app *Application) User(w http.ResponseWriter, r *http.Request) {
	user, err := app.ResolveUserFromRequest(r)
	if err != nil || user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userJson, err := json.Marshal(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(userJson)
}
