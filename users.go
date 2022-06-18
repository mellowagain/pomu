package main

import (
	"database/sql"
	"google.golang.org/api/oauth2/v2"
)

type User struct {
	id     string
	name   string
	avatar string
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

func CreateUser(userInfo *oauth2.Userinfo, db *sql.DB) (*User, error) {
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
