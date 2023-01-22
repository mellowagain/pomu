package main

import (
	"database/sql"
	"github.com/getsentry/sentry-go"
	"net/http"
	"time"
)

type Session struct {
	UserId    string    `json:"userId"`
	Provider  string    `json:"provider"`
	Hash      string    `json:"-"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func FindSession(userId string, provider string, sessionHash string, db *sql.DB) (*User, error) {
	tx, err := db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	defer tx.Rollback()

	statement, err := tx.Prepare(
		`select users.* from sessions 
    	inner join users on sessions.user_id = users.id and sessions.provider = users.provider 
        where sessions.user_id = $1 and sessions.provider = $2 and sessions.hash = $3
        limit 1`)

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	var user User

	if err = statement.QueryRow(userId, provider, sessionHash).Scan(&user.Id, &user.Name, &user.Avatar, &user.Provider); err != nil {
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

func StartSession(userId string, provider string, country string, db *sql.DB) (*Session, error) {
	tx, err := db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	defer tx.Rollback()

	statement, err := tx.Prepare("insert into sessions (user_id, provider, country) values ($1, $2, $3) returning *")

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	var session Session

	if err = statement.QueryRow(userId, provider, country).Scan(&session.UserId, &session.Provider, &session.Hash, &session.Country, &session.CreatedAt, &session.UpdatedAt); err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	return &session, nil
}

func (app *Application) ResolveUserFromRequest(r *http.Request) (user *User, err error) {
	cookie, err := r.Cookie("pomu")

	if err != nil {
		return nil, err
	}

	var session *Session

	if err = app.secureCookie.Decode("session", cookie.Value, &session); err != nil {
		return nil, err
	}

	return FindSession(session.UserId, session.Provider, session.Hash, app.db)
}
