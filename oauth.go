package main

import (
	"context"
	"github.com/hymkor/go-lazy"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googleOauth2 "google.golang.org/api/oauth2/v2"
	youtube "google.golang.org/api/youtube/v3"
	"net/http"
	"os"
)

var oAuthConfig = lazy.New(func() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Scopes:       []string{googleOauth2.UserinfoProfileScope, youtube.YoutubeUploadScope},
	}
})

func OauthLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Google itself does not recommend a `state` (csrf token) so keep it empty
	url := oAuthConfig.Value().AuthCodeURL("")

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func OauthRedirectHandler(w http.ResponseWriter, r *http.Request) {
	if oauthErr := r.URL.Query().Get("oauth_err"); len(oauthErr) <= 0 {
		http.Redirect(w, r, "/?oauthError="+oauthErr, http.StatusTemporaryRedirect)
		return
	}

	code := r.URL.Query().Get("code")
	scope := r.URL.Query().Get("scope")

	if len(code) <= 0 || len(scope) <= 0 {
		http.Error(w, "code and scope cannot be empty", http.StatusBadRequest)
		return
	}

	_, err := oAuthConfig.Value().Exchange(context.Background(), code)

	if err != nil {
		http.Error(w, "failed to exchange code into token. please retry", http.StatusBadGateway)
		return
	}

	/* Create user if not exists; else log them in */
	// Check scopes if they have both, if no upload then require them to always download video
}

/*func CreateOAuth() {
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithTokenSource(""))

}*/
