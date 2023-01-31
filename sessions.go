package main

import (
	"database/sql"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
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

func FindSessionAssociatedUser(userId string, provider string, sessionHash string, db *sql.DB) (*User, error) {
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

func FindSession(userId string, provider string, sessionHash string, db *sql.DB) (*Session, error) {
	tx, err := db.Begin()

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	defer tx.Rollback()

	statement, err := tx.Prepare("select * from sessions where user_id = $1 and provider = $2 and hash = $3 limit 1")

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	var session Session

	if err = statement.QueryRow(userId, provider, sessionHash).Scan(&session.UserId, &session.Provider, &session.Hash, &session.Country, &session.CreatedAt, &session.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			sentry.CaptureException(err)
			return &session, err
		}
	}

	if err = tx.Commit(); err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	return &session, nil
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

func DeleteSession(session *Session, db *sql.DB) error {
	tx, err := db.Begin()

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to begin transaction")
		return err
	}

	defer tx.Rollback()

	_, err = tx.Exec("delete from sessions where user_id = $1 and provider = $2 and hash = $3", session.UserId, session.Provider, session.Hash)

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to execute statement")
		return err
	}

	if tx.Commit() != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to commit transaction")
		return err
	}

	return nil
}

func (app *Application) ResolveUserFromRequest(r *http.Request) (*User, error) {
	cookie, err := r.Cookie("pomu")

	if err != nil {
		return nil, err
	}

	var session *Session

	if err = app.secureCookie.Decode("session", cookie.Value, &session); err != nil {
		return nil, err
	}

	return FindSessionAssociatedUser(session.UserId, session.Provider, session.Hash, app.db)
}

func (app *Application) ResolveSessionFromRequest(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie("pomu")

	if err != nil {
		return nil, err
	}

	var untrustedSession *Session

	if err = app.secureCookie.Decode("session", cookie.Value, &untrustedSession); err != nil {
		return nil, err
	}

	return FindSession(untrustedSession.UserId, untrustedSession.Provider, untrustedSession.Hash, app.db)
}
