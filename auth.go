package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// ---------- Password rules ----------

func isValidPassword(password string) bool {
	hasLetter := false
	hasNumber := false

	for _, char := range password {
		switch {
		case unicode.IsLetter(char):
			hasLetter = true
		case unicode.IsDigit(char):
			hasNumber = true
		}
	}

	return hasLetter && hasNumber
}

// ---------- Who is online? ----------
// We remember the last time each user opened a page.
// "Online" = seen within the last 5 minutes (and not logged out).

const onlineWindow = 5 * time.Minute

var (
	lastSeen   = map[int]time.Time{}
	lastSeenMu sync.Mutex
)

func touch(userID int) {
	lastSeenMu.Lock()
	lastSeen[userID] = time.Now()
	lastSeenMu.Unlock()
}

func isOnline(userID int) bool {
	lastSeenMu.Lock()
	defer lastSeenMu.Unlock()
	t, ok := lastSeen[userID]
	return ok && time.Since(t) < onlineWindow
}

// ---------- Sessions (the "you are logged in" cookie) ----------

const (
	sessionCookie = "session"
	sessionTTL    = 7 * 24 * time.Hour
)

type session struct {
	userID  int
	expires time.Time
}

var (
	sessions   = map[string]session{}
	sessionsMu sync.Mutex
)

func startSession(w http.ResponseWriter, userID int) error {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	token := hex.EncodeToString(b)

	sessionsMu.Lock()
	sessions[token] = session{userID: userID, expires: time.Now().Add(sessionTTL)}
	snapshot := make(map[string]session, len(sessions))
	for k, v := range sessions {
		snapshot[k] = v
	}
	sessionsMu.Unlock()

	// Written to Postgres too, so a redeploy does not sign everyone out.
	if pgOn() {
		if err := pgSaveSessions(snapshot); err != nil {
			log.Println("Could not save the session to the database:", err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // <- turn this on once the site runs on https://
	})
	return nil
}

func endSession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		sessionsMu.Lock()
		if s, ok := sessions[c.Value]; ok {
			lastSeenMu.Lock()
			delete(lastSeen, s.userID) // logged out = offline right away
			lastSeenMu.Unlock()
		}
		delete(sessions, c.Value)
		sessionsMu.Unlock()

		if pgOn() {
			if err := pgDeleteSession(c.Value); err != nil {
				log.Println("Could not clear the session from the database:", err)
			}
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
}

// currentUser returns the logged-in user, or nil for guests.
func currentUser(r *http.Request) *User {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}

	sessionsMu.Lock()
	s, ok := sessions[c.Value]
	if ok && time.Now().After(s.expires) {
		delete(sessions, c.Value)
		ok = false
	}
	sessionsMu.Unlock()
	if !ok {
		return nil
	}

	u, found := findUserByID(s.userID)
	if !found {
		return nil
	}
	touch(u.ID)
	return &u
}

// ---------- Signup ----------

func signupPageHandler(w http.ResponseWriter, r *http.Request) {
	if u := currentUser(r); u != nil {
		http.Redirect(w, r, homeFor(u), http.StatusSeeOther)
		return
	}
	render(w, "signup.html", PageData{})
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	fullName := strings.TrimSpace(r.FormValue("full_name"))
	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	phone := normalisePhone(r.FormValue("phone"))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	// If something is wrong we show the form again with the error
	// and keep what the person already typed (except the password).
	fail := func(msg string) {
		w.WriteHeader(http.StatusBadRequest)
		render(w, "signup.html", PageData{Error: msg, FullName: fullName, Username: username, Email: email, Phone: phone})
	}

	if fullName == "" || username == "" || email == "" || password == "" {
		fail("All fields are required")
		return
	}

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		fail("Please enter a valid email address")
		return
	}

	if phone != "" && (len(phone) < 10 || len(phone) > 14) {
		fail("Please enter a valid phone number, for example 08127758529")
		return
	}
	if phone != "" {
		if _, taken := findUserByPhone(phone); taken {
			fail("That phone number already has an account")
			return
		}
	}

	if password != confirm {
		fail("The two passwords do not match - please type them again")
		return
	}
	if !isValidPassword(password) {
		fail("Password must contain both letters and numbers")
		return
	}
	if len(password) > 72 { // bcrypt only reads the first 72 bytes
		fail("Password is too long (max 72 characters)")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println("Error hashing password:", err)
		return
	}

	newUser, err := createUser(fullName, username, email, string(hashedPassword), "")
	switch {
	case errors.Is(err, ErrUsernameTaken):
		fail("That username is already taken")
		return
	case errors.Is(err, ErrEmailTaken):
		fail("That email is already registered")
		return
	case err != nil:
		log.Println("Error saving user:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if phone != "" && err == nil {
		if phoneErr := setUserPhone(newUser.ID, phone); phoneErr != nil {
			log.Println("Could not save the phone number:", phoneErr)
		}
	}

	// "go" = run in the background, so the user doesn't wait for the messages.
	go welcomeMessages(fullName, email, phone)

	http.Redirect(w, r, "/login?created=1", http.StatusSeeOther)
}

// ---------- Login / Logout ----------

// A fake hash so a wrong username takes as long as a wrong password.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password1"), bcrypt.DefaultCost)

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	if u := currentUser(r); u != nil {
		http.Redirect(w, r, homeFor(u), http.StatusSeeOther)
		return
	}
	render(w, "login.html", PageData{Created: r.URL.Query().Get("created") == "1"})
}

// authenticate checks a username + password. One check, used by both login doors.
// It returns the user and an empty reason on success, or the reason it failed
// (the reason is only written in the server terminal, never shown on the page).
// findAccount accepts a username, an email address or a phone number.
func findAccount(id string) (User, bool) {
	if u, ok := findUserByUsername(id); ok {
		return u, true
	}
	if strings.Contains(id, "@") {
		if u, ok := findUserByEmail(id); ok {
			return u, true
		}
	}
	if u, ok := findUserByPhone(id); ok {
		return u, true
	}
	return User{}, false
}

func authenticate(username, password string) (User, string) {
	user, found := findAccount(username)
	hash := dummyHash
	if found {
		hash = []byte(user.PasswordHash)
	}
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	switch {
	case !found:
		return User{}, "no account with that username, email or number"
	case err != nil:
		return User{}, "wrong password"
	}
	return user, ""
}

// Door 1: the normal login page (everyone, admins included).
func loginHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	user, why := authenticate(username, password)
	if why != "" {
		log.Printf("LOGIN FAILED for %q: %s", username, why)
		w.WriteHeader(http.StatusUnauthorized)
		render(w, "login.html", PageData{Error: "Invalid username or password", Username: username})
		return
	}

	if err := startSession(w, user.ID); err != nil {
		log.Println("Error starting session:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, homeFor(&user), http.StatusSeeOther)
}

// Door 2: the admin login page (correct password AND must be an admin).
func adminLoginPageHandler(w http.ResponseWriter, r *http.Request) {
	if u := currentUser(r); u != nil {
		http.Redirect(w, r, homeFor(u), http.StatusSeeOther)
		return
	}
	render(w, "admin-login.html", PageData{})
}

func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	user, why := authenticate(username, password)
	if why == "" && !user.IsAdmin() {
		why = "this account is not an admin"
	}
	if why != "" {
		log.Printf("ADMIN LOGIN FAILED for %q: %s", username, why)
		// Same message on the page for every reason, so nobody can use this
		// page to find out which usernames are admins.
		w.WriteHeader(http.StatusUnauthorized)
		render(w, "admin-login.html", PageData{Error: "Invalid admin username or password", Username: username})
		return
	}

	if err := startSession(w, user.ID); err != nil {
		log.Println("Error starting session:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// Where to send someone right after they log in.
func homeFor(u *User) string {
	if u.IsAdmin() {
		return "/admin"
	}
	return "/home"
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	endSession(w, r)
	http.Redirect(w, r, "/goodbye", http.StatusSeeOther)
}

// loadSavedSessions restores logins that were still valid when the server last
// stopped. Without this, every restart and every Render sleep would log out
// everyone who was signed in.
func loadSavedSessions() {
	if !pgOn() {
		return
	}
	fresh := map[string]session{}
	if err := pgLoadSessions(fresh); err != nil {
		log.Println("Could not read saved sessions:", err)
		return
	}
	sessionsMu.Lock()
	for k, v := range fresh {
		sessions[k] = v
	}
	sessionsMu.Unlock()
	if len(fresh) > 0 {
		log.Printf("Sessions restored: %d still signed in", len(fresh))
	}
}
