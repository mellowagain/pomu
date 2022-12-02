package main

import (
	"context"
	"database/sql"

	"github.com/getsentry/sentry-go"
	"golang.org/x/oauth2"
	googleOauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type User struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// ResolveUser gets the user using the token
func ResolveUser(token *oauth2.Token, db *sql.DB) (*User, error) {
	service, err := googleOauth2.NewService(context.Background(), option.WithTokenSource(oauth2.StaticTokenSource(token)))

	if err != nil {
		return nil, err
	}

	userInfoService := googleOauth2.NewUserinfoService(service)
	info, err := userInfoService.Get().Do()

	if err != nil {
		return nil, err
	}

	user, err := GetUser(info.Id, db)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func GetUser(id string, db *sql.DB) (*User, error) {
	tx, err := db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	defer tx.Rollback()

	statement, err := tx.Prepare("select * from users where id = $1 limit 1")

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	var user User

	if err = statement.QueryRow(id).Scan(&user.Id, &user.Name, &user.Avatar); err != nil {
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

func CreateUser(userInfo *googleOauth2.Userinfo, db *sql.DB) (*User, error) {
	tx, err := db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	defer tx.Rollback()

	statement, err := tx.Prepare("insert into users (id, name, avatar) values ($1, $2, $3) returning *")

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	var user User

	if err = statement.QueryRow(userInfo.Id, userInfo.GivenName, userInfo.Picture).Scan(&user.Id, &user.Name, &user.Avatar); err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	return &user, nil
}
