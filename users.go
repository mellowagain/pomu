package main

import (
	"database/sql"

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

// ValidateOrCreateUser gets the user based on ID and provider and if they do not exist, registers them. Returns redirect URL
func ValidateOrCreateUser(id string, name string, avatarUrl string, provider string, db *sql.DB) (string, error) {
	user, err := GetUser(id, provider, db)

	if err != nil {
		return "", err
	}

	if user == nil {
		_, err := CreateUser(id, name, avatarUrl, provider, db)

		if err != nil {
			return "", err
		}

		return "/?successRegister", nil
	} else {
		return "/?success", nil
	}
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

func CreateUser(id string, name string, avatarUrl string, provider string, db *sql.DB) (*User, error) {
	tx, err := db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	defer tx.Rollback()

	statement, err := tx.Prepare("insert into users (id, name, avatar, provider) values ($1, $2, $3, $4) returning *")

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
