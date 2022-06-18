package main

import (
	"context"
	"database/sql"
	"golang.org/x/oauth2"
	googleOauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type User struct {
	id     string
	name   string
	avatar string
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
		return nil, err
	}

	statement, err := tx.Prepare("select * from users where id = $1 limit 1")

	var user User

	if err = statement.QueryRow(id).Scan(&user.id, &user.name, &user.avatar); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return &user, err
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &user, nil
}

func CreateUser(userInfo *googleOauth2.Userinfo, db *sql.DB) (*User, error) {
	tx, err := db.Begin()

	if err != nil {
		return nil, err
	}

	statement, err := tx.Prepare("insert into users (id, name, avatar) values ($1, $2, $3) returning *")

	var user User

	if err = statement.QueryRow(userInfo.Id, userInfo.GivenName, userInfo.Picture).Scan(&user.id, &user.name, &user.avatar); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &user, nil
}
