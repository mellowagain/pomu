package main

import (
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

func OauthRedirectHandler(_ http.ResponseWriter, _ *http.Request) {

}

/*func CreateOAuth() {
	ctx := context.Background()
	service, err := youtube.NewService(ctx, option.WithTokenSource(""))

}*/
