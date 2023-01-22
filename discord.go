package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/hymkor/go-lazy"
	"github.com/ravener/discord-oauth2"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
	"os"
)

var discordOAuth = lazy.New(func() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("DISCORD_OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("DISCORD_OAUTH_CLIENT_SECRET"),
		Endpoint:     discord.Endpoint,
		RedirectURL:  os.Getenv("BASE_URL") + "/oauth/discord/redirect",
		Scopes:       []string{discord.ScopeIdentify},
	}
})

func (app *Application) DiscordOAuthInitiator(w http.ResponseWriter, r *http.Request) {
	state := RandomString(16)

	cookie, err := app.secureCookie.Encode("oauth_discord", state)

	if err != nil {
		http.Error(w, "failed to encode", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Set-Cookie", "pomu_oauth="+cookie+"; Path=/; Max-Age=300; HttpOnly")

	url := discordOAuth.Value().AuthCodeURL(state)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (app *Application) DiscordOAuthRedirect(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("pomu_oauth")

	if err != nil {
		http.Error(w, "no csrf token", http.StatusBadRequest)
		return
	}

	w.Header().Set("Set-Cookie", "pomu_oauth=deleted; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT")

	var csrfToken string

	err = app.secureCookie.Decode("oauth_discord", cookie.Value, &csrfToken)

	if err != nil {
		http.Error(w, "failed to decode csrf token", http.StatusInternalServerError)
		return
	}

	state := r.FormValue("state")

	if csrfToken != state {
		http.Error(w, "csrf token mismatch", http.StatusBadRequest)
		return
	}

	token, err := discordOAuth.Value().Exchange(context.Background(), r.FormValue("code"))

	if err != nil {
		http.Error(w, "failed to exchange token", http.StatusBadGateway)
		sentry.CaptureException(err)
		return
	}

	id, name, avatarUrl, err := resolveUserWithDiscordToken(token)

	if err != nil {
		http.Error(w, "failed to get discord info", http.StatusBadGateway)
		sentry.CaptureException(err)
		return
	}

	redirectUrl, err := ValidateOrCreateUser(id, name, avatarUrl, ProviderDiscord, app.db)

	if err != nil {
		http.Error(w, "failed to get or create user", http.StatusInternalServerError)
		sentry.CaptureException(err)
		return
	}

	session, err := StartSession(id, ProviderDiscord, r.Header.Get("CF-IPCountry"), app.db)

	if err != nil {
		http.Error(w, "failed to start session", http.StatusInternalServerError)
		sentry.CaptureException(err)
		return
	}

	encodedCookie, err := app.secureCookie.Encode("session", session)

	if err != nil {
		http.Error(w, "failed to encode cookie", http.StatusInternalServerError)
		sentry.CaptureException(err)
		return
	}

	// 604'800 = a week
	w.Header().Set("Set-Cookie", "pomu="+encodedCookie+"; Path=/; Max-Age=604800")

	http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
}

func resolveUserWithDiscordToken(token *oauth2.Token) (string, string, string, error) {
	response, err := discordOAuth.Value().Client(context.Background(), token).Get("https://discord.com/api/users/@me")

	if err != nil || response.StatusCode != 200 {
		return "", "", "", err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return "", "", "", err
	}

	var responses map[string]any

	if err := json.Unmarshal(body, &responses); err != nil {
		return "", "", "", err
	}

	id := responses["id"].(string)

	username := responses["username"].(string)
	discriminator := responses["discriminator"].(string)

	name := fmt.Sprintf("%s#%s", username, discriminator)

	avatarHash := responses["avatar"].(string)
	avatarUrl := fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", id, avatarHash)

	return id, name, avatarUrl, nil
}
