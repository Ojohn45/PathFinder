package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// ---------- Where the data really lives ----------
//
// PathFinder used to keep everything in JSON files under storage/. On Render the
// disk is wiped on every redeploy, restart and sleep, so accounts and history
// vanished. That was the bug, not the logout button: logout only deletes the
// session cookie.
//
// Now, if DATABASE_URL is set, every collection is stored in Postgres and the
// JSON files are only used on a developer's laptop. The rest of the app is
// unchanged: it still works with the in-memory slices, and each change is
// written through to Postgres before the request finishes.
//
// The connection string comes from the environment only. No secrets in code.

var db *sql.DB

// pgOn reports whether a database is configured.
func pgOn() bool { return db != nil }

// initStore connects to Postgres when DATABASE_URL exists, and creates the
// tables if they are not there yet. It is safe to run on every start.
func initStore() error {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("PG_URL"))
	}
	if dsn == "" {
		log.Println("STORAGE: DATABASE_URL not set. Falling back to JSON files in storage/.")
		log.Println("STORAGE: on Render those files disappear on redeploy. Set DATABASE_URL to keep data.")
		return nil
	}

	// Hosted Postgres (Neon, Supabase, Render) needs TLS. Add it unless the
	// address already says how to connect.
	if !strings.Contains(dsn, "sslmode=") {
		if strings.Contains(dsn, "?") {
			dsn += "&sslmode=require"
		} else {
			dsn += "?sslmode=require"
		}
	}

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(8)
	conn.SetMaxIdleConns(4)
	conn.SetConnMaxLifetime(time.Hour)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return fmt.Errorf("cannot reach the database: %w", err)
	}

	db = conn
	if err := pgMigrate(); err != nil {
		return fmt.Errorf("cannot create tables: %w", err)
	}
	log.Println("STORAGE: using Postgres. Accounts, history and settings survive redeploys.")
	return nil
}

// pgMigrate creates the tables. Every statement is "if not exists", so running
// it on each start changes nothing after the first time.
func pgMigrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            BIGINT PRIMARY KEY,
			full_name     TEXT NOT NULL DEFAULT '',
			username      TEXT NOT NULL,
			email         TEXT NOT NULL DEFAULT '',
			phone         TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL,
			role          TEXT NOT NULL DEFAULT '',
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		// Avatar and profile columns. ADD COLUMN IF NOT EXISTS means an existing
		// database simply gains them on the next start, losing nothing.
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_kind TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_ref TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_data BYTEA`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_at TIMESTAMPTZ`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS frame TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS banner TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS calm_mode BOOLEAN NOT NULL DEFAULT false`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower ON users (lower(username))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_email_lower ON users (lower(email)) WHERE email <> ''`,

		`CREATE TABLE IF NOT EXISTS history (
			id         BIGSERIAL PRIMARY KEY,
			user_id    BIGINT NOT NULL,
			clicked_at TIMESTAMPTZ NOT NULL,
			payload    JSONB NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS history_user_time ON history (user_id, clicked_at DESC)`,

		`CREATE TABLE IF NOT EXISTS invites (
			token_hash TEXT PRIMARY KEY,
			payload    JSONB NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS email_log (
			id       BIGSERIAL PRIMARY KEY,
			sent_at  TIMESTAMPTZ NOT NULL,
			to_addr  TEXT NOT NULL DEFAULT '',
			subject  TEXT NOT NULL DEFAULT '',
			status   TEXT NOT NULL DEFAULT '',
			err_text TEXT NOT NULL DEFAULT ''
		)`,

		`CREATE TABLE IF NOT EXISTS settings (
			key        TEXT PRIMARY KEY,
			value      JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,

		`CREATE TABLE IF NOT EXISTS sessions (
			token   TEXT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			expires TIMESTAMPTZ NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS sessions_expires ON sessions (expires)`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("%w\nin: %s", err, s)
		}
	}
	return nil
}

// pqInt64Array renders ids as a Postgres array literal for "= ANY($1)".
func pqInt64Array(ids []int64) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%d", id)
	}
	b.WriteByte('}')
	return b.String()
}

// ---------- Loading ----------

// pgLoad fills v from Postgres. The path is the same one the JSON code used,
// so callers do not change.
func pgLoad(path string, v any) error {
	switch path {
	case usersFile:
		return pgLoadUsers(v.(*[]User))
	case historyFile:
		return pgLoadHistory(v.(*[]HistoryEntry))
	case invitesFile:
		return pgLoadInvites(v.(*[]Invite))
	case emailsFile:
		return pgLoadEmails(v.(*[]EmailRecord))
	}
	return errors.New("no table for " + path)
}

func pgLoadUsers(dst *[]User) error {
	rows, err := db.Query(`SELECT id, full_name, username, email, phone, password_hash, role, created_at,
			display_name, bio, avatar_kind, avatar_ref, avatar_data, avatar_at, frame, banner, calm_mode
		FROM users ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	out := []User{}
	for rows.Next() {
		var u User
		var avatarAt sql.NullTime
		var avatarData []byte
		if err := rows.Scan(&u.ID, &u.FullName, &u.Username, &u.Email, &u.Phone,
			&u.PasswordHash, &u.Role, &u.CreatedAt,
			&u.DisplayName, &u.Bio, &u.AvatarKind, &u.AvatarRef, &avatarData, &avatarAt,
			&u.Frame, &u.Banner, &u.CalmMode); err != nil {
			return err
		}
		u.AvatarData = avatarData
		if avatarAt.Valid {
			u.AvatarAt = avatarAt.Time
		}
		out = append(out, u)
	}
	*dst = out
	return rows.Err()
}

func pgLoadHistory(dst *[]HistoryEntry) error {
	rows, err := db.Query(`SELECT payload FROM history ORDER BY clicked_at`)
	if err != nil {
		return err
	}
	defer rows.Close()

	out := []HistoryEntry{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		var e HistoryEntry
		if err := json.Unmarshal(raw, &e); err != nil {
			return err
		}
		out = append(out, e)
	}
	*dst = out
	return rows.Err()
}

func pgLoadInvites(dst *[]Invite) error {
	rows, err := db.Query(`SELECT payload FROM invites`)
	if err != nil {
		return err
	}
	defer rows.Close()

	out := []Invite{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		var v Invite
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}
		out = append(out, v)
	}
	*dst = out
	return rows.Err()
}

func pgLoadEmails(dst *[]EmailRecord) error {
	rows, err := db.Query(`SELECT sent_at, to_addr, subject, status, err_text
		FROM email_log ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	out := []EmailRecord{}
	for rows.Next() {
		var e EmailRecord
		if err := rows.Scan(&e.Time, &e.To, &e.Subject, &e.Status, &e.Error); err != nil {
			return err
		}
		out = append(out, e)
	}
	*dst = out
	return rows.Err()
}

// ---------- Saving ----------

// pgSave writes a whole collection. The app keeps its data in memory and hands
// the current list here, so nothing else has to change.
func pgSave(path string, v any) error {
	switch path {
	case usersFile:
		return pgSyncUsers(v.([]User))
	case historyFile:
		return pgSyncHistory(v.([]HistoryEntry))
	case invitesFile:
		return pgSyncInvites(v.([]Invite))
	case emailsFile:
		return pgSyncEmails(v.([]EmailRecord))
	}
	return errors.New("no table for " + path)
}

// pgSyncUsers upserts every account and removes any that were deleted, in one
// transaction. Accounts are never dropped by a redeploy.
func pgSyncUsers(list []User) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ids := make([]int64, 0, len(list))
	for _, u := range list {
		ids = append(ids, int64(u.ID))
		if _, err := tx.Exec(`INSERT INTO users
				(id, full_name, username, email, phone, password_hash, role, created_at,
				 display_name, bio, avatar_kind, avatar_ref, avatar_data, avatar_at,
				 frame, banner, calm_mode)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
			ON CONFLICT (id) DO UPDATE SET
				full_name = EXCLUDED.full_name,
				username  = EXCLUDED.username,
				email     = EXCLUDED.email,
				phone     = EXCLUDED.phone,
				password_hash = EXCLUDED.password_hash,
				role      = EXCLUDED.role,
				display_name = EXCLUDED.display_name,
				bio          = EXCLUDED.bio,
				avatar_kind  = EXCLUDED.avatar_kind,
				avatar_ref   = EXCLUDED.avatar_ref,
				avatar_data  = EXCLUDED.avatar_data,
				avatar_at    = EXCLUDED.avatar_at,
				frame        = EXCLUDED.frame,
				banner       = EXCLUDED.banner,
				calm_mode    = EXCLUDED.calm_mode`,
			u.ID, u.FullName, u.Username, u.Email, u.Phone, u.PasswordHash, u.Role, u.CreatedAt,
			u.DisplayName, u.Bio, u.AvatarKind, u.AvatarRef, u.AvatarData, u.AvatarAt,
			u.Frame, u.Banner, u.CalmMode); err != nil {
			return err
		}
	}

	if len(ids) == 0 {
		if _, err := tx.Exec(`DELETE FROM users`); err != nil {
			return err
		}
	} else if _, err := tx.Exec(`DELETE FROM users WHERE NOT (id = ANY($1))`, pqInt64Array(ids)); err != nil {
		return err
	}
	return tx.Commit()
}

func pgSyncHistory(list []HistoryEntry) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM history`); err != nil {
		return err
	}
	for _, e := range list {
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		when := e.ClickedAt
		if when.IsZero() {
			when = time.Now()
		}
		if _, err := tx.Exec(`INSERT INTO history (user_id, clicked_at, payload) VALUES ($1,$2,$3)`,
			e.UserID, when, string(raw)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func pgSyncInvites(list []Invite) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM invites`); err != nil {
		return err
	}
	for _, v := range list {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO invites (token_hash, payload) VALUES ($1,$2)
			ON CONFLICT (token_hash) DO UPDATE SET payload = EXCLUDED.payload`, v.TokenHash, string(raw)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func pgSyncEmails(list []EmailRecord) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM email_log`); err != nil {
		return err
	}
	for _, e := range list {
		when := e.Time
		if when.IsZero() {
			when = time.Now()
		}
		if _, err := tx.Exec(`INSERT INTO email_log (sent_at, to_addr, subject, status, err_text)
			VALUES ($1,$2,$3,$4,$5)`, when, e.To, e.Subject, e.Status, e.Error); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---------- Settings (mail details and the tab icon) ----------

const settingsKey = "mail"

func storeReadSettings() ([]byte, error) {
	if pgOn() {
		var raw []byte
		err := db.QueryRow(`SELECT value FROM settings WHERE key = $1`, settingsKey).Scan(&raw)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return raw, err
	}
	return os.ReadFile(settingsPath)
}

func storeWriteSettings(data []byte) error {
	if pgOn() {
		_, err := db.Exec(`INSERT INTO settings (key, value, updated_at) VALUES ($1,$2,now())
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
			settingsKey, string(data))
		return err
	}
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(settingsPath, data, 0o600)
}

// ---------- Sessions ----------
//
// Sessions live in Postgres too, so a redeploy or a sleep does not log everyone
// out. The cookie stays in the browser; the server only checks the table.

func pgLoadSessions(into map[string]session) error {
	rows, err := db.Query(`SELECT token, user_id, expires FROM sessions WHERE expires > now()`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			token   string
			userID  int
			expires time.Time
		)
		if err := rows.Scan(&token, &userID, &expires); err != nil {
			return err
		}
		into[token] = session{userID: userID, expires: expires}
	}
	return rows.Err()
}

func pgSaveSessions(all map[string]session) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sessions WHERE expires <= now()`); err != nil {
		return err
	}
	for token, s := range all {
		if s.expires.Before(time.Now()) {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO sessions (token, user_id, expires) VALUES ($1,$2,$3)
			ON CONFLICT (token) DO UPDATE SET user_id = EXCLUDED.user_id, expires = EXCLUDED.expires`,
			token, s.userID, s.expires); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func pgDeleteSession(token string) error {
	if !pgOn() {
		return nil
	}
	_, err := db.Exec(`DELETE FROM sessions WHERE token = $1`, token)
	return err
}
