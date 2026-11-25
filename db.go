package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ---------- What we store ----------

type User struct {
	ID           int       `json:"id"`
	FullName     string    `json:"full_name"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone,omitempty"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role,omitempty"` // "" = normal user, "admin", "head_admin"
	CreatedAt    time.Time `json:"created_at"`
}

const (
	RoleAdmin     = "admin"
	RoleHeadAdmin = "head_admin"
)

func (u User) IsAdmin() bool     { return u.Role == RoleAdmin || u.Role == RoleHeadAdmin }
func (u User) IsHeadAdmin() bool { return u.Role == RoleHeadAdmin }

func (u User) RoleLabel() string {
	switch u.Role {
	case RoleHeadAdmin:
		return "Head Admin"
	case RoleAdmin:
		return "Admin"
	}
	return "User"
}

// One line in the email diary: what the server did with an email.
type EmailRecord struct {
	Time    time.Time `json:"time"`
	To      string    `json:"to"`
	Subject string    `json:"subject"`
	Status  string    `json:"status"` // "sent", "failed" or "not_configured"
	Error   string    `json:"error,omitempty"`
}

// An invitation for someone to become an admin.
// We only keep a HASH of the secret token, like a password.
type Invite struct {
	TokenHash string    `json:"token_hash"`
	Email     string    `json:"email"`
	InvitedBy int       `json:"invited_by"`
	ForUser   bool      `json:"for_user,omitempty"` // true = invite a normal user; false = invite an admin
	CreatedAt time.Time `json:"created_at"`
}

type HistoryEntry struct {
	UserID       int       `json:"user_id"`
	LanguageSlug string    `json:"language_slug"`
	ResourceID   string    `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	ClickedAt    time.Time `json:"clicked_at"`

	// What kind of step this was: "" (a place opened), "search" (a query typed)
	// or "result" (a search result that was clicked).
	Kind        string `json:"kind,omitempty"`
	Query       string `json:"query,omitempty"`
	SectionSlug string `json:"section_slug,omitempty"`
	DestURL     string `json:"dest_url,omitempty"`
	DestName    string `json:"dest_name,omitempty"`
}

// ---------- Where we store it ----------

const storageDir = "storage"

var (
	usersFile   = filepath.Join(storageDir, "users.json")
	historyFile = filepath.Join(storageDir, "history.json")
	invitesFile = filepath.Join(storageDir, "invites.json")
	emailsFile  = filepath.Join(storageDir, "emails.json")

	// One lock protects both slices below, so two requests
	// can never write to the files at the same moment.
	mu       sync.Mutex
	users    []User
	history  []HistoryEntry
	invites  []Invite
	emailLog []EmailRecord
)

var (
	ErrUsernameTaken = errors.New("username already taken")
	ErrEmailTaken    = errors.New("email already registered")
)

// initDB creates the storage folder and loads the JSON files into memory.
func initDB() {
	if err := initStore(); err != nil {
		log.Fatal("Database problem:", err)
	}
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		log.Fatal("Cannot create storage folder:", err)
	}
	if err := loadJSON(usersFile, &users); err != nil {
		log.Fatal("Cannot read users.json:", err)
	}
	if err := loadJSON(historyFile, &history); err != nil {
		log.Fatal("Cannot read history.json:", err)
	}
	if err := loadJSON(invitesFile, &invites); err != nil {
		log.Fatal("Cannot read invites.json:", err)
	}
	if err := loadJSON(emailsFile, &emailLog); err != nil {
		log.Fatal("Cannot read emails.json:", err)
	}
	loadSavedSessions()
	seedHeadAdmin()
	log.Printf("Storage ready: %d user(s), %d history entries", len(users), len(history))
}

// loadJSON reads a file into v. A missing file is fine (first run).
func loadJSON(path string, v any) error {
	if pgOn() {
		return pgLoad(path, v)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// saveJSON writes to a temp file first, then renames it over the real one.
// That way a crash mid-write can never leave a half-written (corrupt) file.
func saveJSON(path string, v any) error {
	if pgOn() {
		return pgSave(path, v)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ---------- Users ----------

func createUser(fullName, username, email, passwordHash, role string) (User, error) {
	mu.Lock()
	defer mu.Unlock()

	nextID := 1
	for _, u := range users {
		if strings.EqualFold(u.Username, username) {
			return User{}, ErrUsernameTaken
		}
		if strings.EqualFold(u.Email, email) {
			return User{}, ErrEmailTaken
		}
		if u.ID >= nextID {
			nextID = u.ID + 1
		}
	}

	u := User{
		ID:           nextID,
		FullName:     fullName,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    time.Now(),
	}
	users = append(users, u)

	if err := saveJSON(usersFile, users); err != nil {
		users = users[:len(users)-1] // undo, so memory matches the file
		return User{}, err
	}
	return u, nil
}

func findUserByUsername(username string) (User, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, u := range users {
		if strings.EqualFold(u.Username, username) {
			return u, true
		}
	}
	return User{}, false
}

// normalisePhone turns +2348127758529, 2348127758529 and 0812 775 8529 into
// 08127758529, so the head admin can log in however they type the number.
func normalisePhone(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	switch {
	case strings.HasPrefix(digits, "234") && len(digits) == 13:
		return "0" + digits[3:]
	case len(digits) == 10 && digits[0] != '0':
		return "0" + digits
	}
	return digits
}

func findUserByPhone(phone string) (User, bool) {
	want := normalisePhone(phone)
	if want == "" {
		return User{}, false
	}
	mu.Lock()
	defer mu.Unlock()
	for _, u := range users {
		if u.Phone != "" && normalisePhone(u.Phone) == want {
			return u, true
		}
	}
	return User{}, false
}

// setUserPhone saves a phone number on an account (used for the head admin).
func setUserPhone(id int, phone string) error {
	mu.Lock()
	defer mu.Unlock()
	for i := range users {
		if users[i].ID == id {
			old := users[i].Phone
			users[i].Phone = phone
			if err := saveJSON(usersFile, users); err != nil {
				users[i].Phone = old
				return err
			}
			return nil
		}
	}
	return errors.New("no such user")
}

func findUserByID(id int) (User, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, u := range users {
		if u.ID == id {
			return u, true
		}
	}
	return User{}, false
}

// ---------- History ----------

// addEvent records anything worth remembering: a place opened, a search typed,
// or a search result followed.
func addEvent(e HistoryEntry) error {
	mu.Lock()
	defer mu.Unlock()
	if e.ClickedAt.IsZero() {
		e.ClickedAt = time.Now()
	}
	history = append(history, e)
	if err := saveJSON(historyFile, history); err != nil {
		history = history[:len(history)-1]
		return err
	}
	return nil
}

func addHistory(userID int, slug, resourceID, resourceName string) error {
	return addEvent(HistoryEntry{
		UserID:       userID,
		LanguageSlug: slug,
		ResourceID:   resourceID,
		ResourceName: resourceName,
	})
}

func addSearchEvent(userID int, sectionSlug, query string) error {
	return addEvent(HistoryEntry{
		UserID:       userID,
		Kind:         "search",
		SectionSlug:  sectionSlug,
		Query:        query,
		ResourceName: query,
	})
}

func addResultEvent(userID int, sectionSlug, query, destName, destURL string) error {
	return addEvent(HistoryEntry{
		UserID:       userID,
		Kind:         "result",
		SectionSlug:  sectionSlug,
		Query:        query,
		DestName:     destName,
		DestURL:      destURL,
		ResourceName: destName,
	})
}

// recentHistory returns the user's last n visits, newest first.
func recentHistory(userID, n int) []HistoryEntry {
	mu.Lock()
	defer mu.Unlock()
	var out []HistoryEntry
	for i := len(history) - 1; i >= 0 && len(out) < n; i-- {
		if history[i].UserID == userID {
			out = append(out, history[i])
		}
	}
	return out
}

// ---------- More user helpers ----------

func findUserByEmail(email string) (User, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, u := range users {
		if strings.EqualFold(u.Email, email) {
			return u, true
		}
	}
	return User{}, false
}

// listUsers returns a copy of every user (oldest first).
func listUsers() []User {
	mu.Lock()
	defer mu.Unlock()
	out := make([]User, len(users))
	copy(out, users)
	return out
}

func headAdmin() (User, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, u := range users {
		if u.Role == RoleHeadAdmin {
			return u, true
		}
	}
	return User{}, false
}

func setUserRole(id int, role string) error {
	mu.Lock()
	defer mu.Unlock()
	for i := range users {
		if users[i].ID == id {
			old := users[i].Role
			users[i].Role = role
			if err := saveJSON(usersFile, users); err != nil {
				users[i].Role = old
				return err
			}
			return nil
		}
	}
	return errors.New("user not found")
}

// ---------- Admin invites ----------

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// createInvite makes a secret token for this email. Older links for the
// same email keep working. The token is returned ONCE (to put in the email
// link); only its hash is saved.
func createInvite(email string, invitedBy int, forUser bool) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	mu.Lock()
	defer mu.Unlock()

	invites = append(invites, Invite{
		TokenHash: hashToken(token),
		Email:     email,
		InvitedBy: invitedBy,
		ForUser:   forUser,
		CreatedAt: time.Now(),
	})
	if err := saveJSON(invitesFile, invites); err != nil {
		invites = invites[:len(invites)-1]
		return "", err
	}
	return token, nil
}

// findInvite finds the invite for a secret link. Links never expire and can
// be opened as many times as you like.
func findInvite(token string) (Invite, bool) {
	h := hashToken(token)
	mu.Lock()
	defer mu.Unlock()
	for _, inv := range invites {
		if inv.TokenHash == h {
			return inv, true
		}
	}
	return Invite{}, false
}

// promoteToHeadAdmin turns an EXISTING account into the head admin and
// sets its name, username and password hash. Used when the server starts.
func promoteToHeadAdmin(id int, fullName, username, passwordHash string) error {
	mu.Lock()
	defer mu.Unlock()

	idx := -1
	for i, u := range users {
		if u.ID == id {
			idx = i
		} else if strings.EqualFold(u.Username, username) {
			return ErrUsernameTaken // some OTHER account already uses that username
		}
	}
	if idx < 0 {
		return errors.New("user not found")
	}

	old := users[idx]
	users[idx].FullName = fullName
	users[idx].Username = username
	users[idx].PasswordHash = passwordHash
	users[idx].Role = RoleHeadAdmin

	if err := saveJSON(usersFile, users); err != nil {
		users[idx] = old
		return err
	}
	return nil
}

// ---------- Email diary ----------

const maxEmailRecords = 300 // keep the newest 300 only

func addEmailRecord(rec EmailRecord) {
	mu.Lock()
	defer mu.Unlock()
	emailLog = append(emailLog, rec)
	if len(emailLog) > maxEmailRecords {
		emailLog = emailLog[len(emailLog)-maxEmailRecords:]
	}
	if err := saveJSON(emailsFile, emailLog); err != nil {
		log.Println("Could not save the email diary:", err)
	}
}

// recentEmails returns the newest n records, newest first.
func recentEmails(n int) []EmailRecord {
	mu.Lock()
	defer mu.Unlock()
	var out []EmailRecord
	for i := len(emailLog) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, emailLog[i])
	}
	return out
}

// setUserPassword replaces one user's password hash.
func setUserPassword(id int, passwordHash string) error {
	mu.Lock()
	defer mu.Unlock()
	for i := range users {
		if users[i].ID == id {
			old := users[i].PasswordHash
			users[i].PasswordHash = passwordHash
			if err := saveJSON(usersFile, users); err != nil {
				users[i].PasswordHash = old
				return err
			}
			return nil
		}
	}
	return errors.New("user not found")
}
