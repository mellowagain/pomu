package main

import (
	"database/sql"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/getsentry/sentry-go"
)

const (
	ProviderGoogle  = "google"
	ProviderDiscord = "discord"
)

type User struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Provider string `json:"provider"`
}

func (app *Application) IdentitySelf(w http.ResponseWriter, r *http.Request) {
	user, err := app.ResolveUserFromRequest(r)

	if err != nil || user == nil {
		http.Error(w, "please login first", http.StatusUnauthorized)
		return
	}

	SerializeJson(w, user)
}

func (app *Application) Identity(w http.ResponseWriter, r *http.Request) {
	variables := mux.Vars(r)
	requestedProvider := strings.ToLower(variables["provider"])
	id := strings.ToLower(variables["id"])

	if len(strings.Trim(requestedProvider, " ")) == 0 || len(strings.Trim(id, " ")) == 0 {
		http.Error(w, "provider or id empty", http.StatusBadRequest)
		return
	}

	var provider string

	switch requestedProvider {
	case "google":
		provider = ProviderGoogle
		break
	case "discord":
		provider = ProviderDiscord
		break
	default:
		http.Error(w, "provider not found", http.StatusBadRequest)
		return
	}

	log.Printf("checking %s provid %s\n", id, requestedProvider)

	user, err := GetUser(id, provider, app.db)

	if err != nil || user == nil {
		log.Printf("err: %s, user %v", err, user)
		http.Error(w, "requested user not found", http.StatusNotFound)
		return
	}

	SerializeJson(w, user)
}

func (app *Application) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := app.ResolveSessionFromRequest(r)

	if err == nil {
		_ = DeleteSession(session, app.db)
	}

	w.Header().Set("Set-Cookie", "pomu=deleted; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT")
	http.Redirect(w, r, "/?successLogout", http.StatusTemporaryRedirect)
}

// ValidateOrCreateUser gets the user based on ID and provider and if they do not exist, registers them. Returns redirect URL
func ValidateOrCreateUser(id string, name string, avatarUrl string, provider string, db *sql.DB) (string, error) {
	user, err := GetUser(id, provider, db)

	if err != nil {
		return "", err
	}

	var redirectUrl string

	if user == nil {
		redirectUrl = "/?successRegister"
	} else {
		redirectUrl = "/?success"
	}

	_, err = CreateOrUpdateUser(id, name, avatarUrl, provider, db)

	if err != nil {
		return "", err
	}

	return redirectUrl, nil
}

func GetUser(id string, provider string, db *sql.DB) (*User, error) {
	tx, err := db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	defer tx.Rollback()

	statement, err := tx.Prepare("select * from users where id = $1 and provider = $2 limit 1")

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	var user User

	if err = statement.QueryRow(id, provider).Scan(&user.Id, &user.Name, &user.Avatar, &user.Provider); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			sentry.CaptureException(err)
			return &user, err
		}
	}

	if err = tx.Commit(); err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	return &user, nil
}

func CreateOrUpdateUser(id string, name string, avatarUrl string, provider string, db *sql.DB) (*User, error) {
	tx, err := db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	defer tx.Rollback()

	statement, err := tx.Prepare(`
		insert into users (id, name, avatar, provider)
		values ($1, $2, $3, $4)
		on conflict (id) do update set
			name = $2,
			avatar = $3
		returning *
`)

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	var user User

	if err = statement.QueryRow(id, name, avatarUrl, provider).Scan(&user.Id, &user.Name, &user.Avatar, &user.Provider); err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	return &user, nil
}
