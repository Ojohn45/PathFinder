package main

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ---------- The head admin ----------
//
// The head admin is created automatically when the server starts, so you can
// log in and use the admin page immediately.
//
// The password has a built-in default below. Override it (recommended before
// this goes online) with an environment variable:
//
//   ADMIN_PASSWORD='a-new-password'  go run .
//
// The other details have defaults below, and can be changed with
// ADMIN_EMAIL, ADMIN_FULLNAME and ADMIN_USERNAME.
//
// If there is NO head admin yet, one is created. If an account with the admin
// email already exists, it is upgraded to head admin. After that, ADMIN_PASSWORD
// in .env is always the head admin's password.

// defaultHeadAdminPassword is used only when there is no head admin yet and
// ADMIN_PASSWORD is not set. Change it here, or set ADMIN_PASSWORD.
const defaultHeadAdminPassword = "49321111@John"

func seedHeadAdmin() {
	if head, exists := headAdmin(); exists {
		// The head admin already exists. If ADMIN_PASSWORD in .env is different from
		// the saved password, .env wins - so changing it there really changes it.
		if pw := os.Getenv("ADMIN_PASSWORD"); pw != "" &&
			bcrypt.CompareHashAndPassword([]byte(head.PasswordHash), []byte(pw)) != nil {
			hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
			if err != nil {
				log.Println("Could not update the head admin password:", err, "(passwords can be at most 72 characters)")
			} else if err := setUserPassword(head.ID, string(hash)); err != nil {
				log.Println("Could not update the head admin password:", err)
			} else {
				log.Printf("Head admin %q: password updated from ADMIN_PASSWORD in .env", head.Username)
			}
		}
		return
	}

	password := getenv("ADMIN_PASSWORD", defaultHeadAdminPassword)

	email := strings.ToLower(getenv("ADMIN_EMAIL", "brucenic12@gmail.com"))
	fullName := getenv("ADMIN_FULLNAME", "Odumu John")
	username := getenv("ADMIN_USERNAME", "Ken")

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Could not create head admin:", err)
		return
	}
	// If this email already has a normal account (for example from an earlier
	// test signup), upgrade THAT account instead of failing.
	if existing, found := findUserByEmail(email); found {
		if err := promoteToHeadAdmin(existing.ID, fullName, username, string(hash)); err != nil {
			log.Println("Could not make", existing.Username, "the head admin:", err, "(another account may already use the username", username+")")
			return
		}
		log.Printf("Existing account %q upgraded to head admin. Log in as %q with your ADMIN_PASSWORD.", existing.Username, username)
		return
	}

	if _, err := createUser(fullName, username, email, string(hash), RoleHeadAdmin); err != nil {
		log.Println("Could not create head admin:", err, "(is that username/email already used by another account?)")
		return
	}
	log.Println("Head admin created:", username)
}

// ---------- Guards: who may open which page ----------

// requireAdmin returns the user if they are an admin.
// Otherwise it redirects them away and returns nil.
func requireAdmin(w http.ResponseWriter, r *http.Request) *User {
	u := currentUser(r)
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	}
	if !u.IsAdmin() {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return nil
	}
	return u
}

// Only the head admin may invite new admins.
func requireHeadAdmin(w http.ResponseWriter, r *http.Request) *User {
	u := requireAdmin(w, r)
	if u == nil {
		return nil
	}
	if !u.IsHeadAdmin() {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return nil
	}
	return u
}

// ---------- Admin pages ----------

func adminPageHandler(w http.ResponseWriter, r *http.Request) {
	u := requireAdmin(w, r)
	if u == nil {
		return
	}
	head, _ := headAdmin()
	render(w, "admin.html", PageData{User: u, HeadAdmin: &head, NoBack: true})
}

func inviteAdminPageHandler(w http.ResponseWriter, r *http.Request) {
	u := requireHeadAdmin(w, r)
	if u == nil {
		return
	}
	render(w, "admin-invite.html", PageData{User: u})
}

func inviteUserPageHandler(w http.ResponseWriter, r *http.Request) {
	u := requireAdmin(w, r) // ANY admin may invite normal users
	if u == nil {
		return
	}
	render(w, "user-invite.html", PageData{User: u})
}

// baseURL is the address used inside emails. It does not need a real domain:
// BASE_URL wins, then Render's own RENDER_EXTERNAL_URL (set automatically for
// every Render service), then localhost for development.
func baseURL() string {
	if v := strings.TrimRight(getenv("BASE_URL", ""), "/"); v != "" {
		return v
	}
	if v := strings.TrimRight(os.Getenv("RENDER_EXTERNAL_URL"), "/"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

// baseURLFor does the same, but when nothing is configured it borrows the
// address the admin is actually using. So an invite made from a phone on the
// same Wi-Fi gets a link that works on that phone, not "localhost".
func baseURLFor(r *http.Request) string {
	if v := strings.TrimRight(getenv("BASE_URL", ""), "/"); v != "" {
		return v
	}
	if v := strings.TrimRight(os.Getenv("RENDER_EXTERNAL_URL"), "/"); v != "" {
		return v
	}
	host := r.Host
	if host == "" {
		return "http://localhost:8080"
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + host
}

// Only the head admin may invite new ADMINS.
func inviteAdminHandler(w http.ResponseWriter, r *http.Request) {
	u := requireHeadAdmin(w, r)
	if u == nil {
		return
	}
	processInvite(w, r, u, false)
}

// Any admin may invite normal USERS.
func inviteUserHandler(w http.ResponseWriter, r *http.Request) {
	u := requireAdmin(w, r)
	if u == nil {
		return
	}
	processInvite(w, r, u, true)
}

// processInvite does the work for both kinds of invite.
func processInvite(w http.ResponseWriter, r *http.Request, u *User, forUser bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	page := "admin-invite.html"
	path := "/admin/accept"
	if forUser {
		page = "user-invite.html"
		path = "/join"
	}

	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	fail := func(msg string) {
		w.WriteHeader(http.StatusBadRequest)
		render(w, page, PageData{User: u, Error: msg, Email: email})
	}

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		fail("Please enter a valid email address")
		return
	}
	if existing, ok := findUserByEmail(email); ok {
		if forUser {
			fail("That person already has an account on PathFinder")
			return
		}
		if existing.IsAdmin() {
			fail("That person is already an admin")
			return
		}
	}

	token, err := createInvite(email, u.ID, forUser)
	if err != nil {
		log.Println("Could not save invite:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	link := baseURLFor(r) + path + "?token=" + token

	var subject, heading, message string
	if forUser {
		subject = "You're invited to join PathFinder 🎉"
		heading = "You're invited to join PathFinder"
		message = "PathFinder would like you to join as a user and start studying programming. " +
			"Why join? It's free, it saves you hours of searching, and it points you straight to 10 of the best places " +
			"to learn any programming language: official docs, interactive tutorials, full courses and practice sites. " +
			"Your account also remembers every resource you've visited, so you never lose your place. " +
			"Would you like to join us?"
	} else {
		subject = "You're invited to be an admin on PathFinder"
		heading = "Join the PathFinder team as an admin"
		message = "PathFinder would like you to join the team as an admin. " +
			"As an admin, you'll help look after our community of learners and help the site grow. " +
			"PathFinder is free and saves learners hours of searching by pointing them straight to 15 of the best places " +
			"to learn any programming language. " +
			"Would you like to join the team?"
	}
	text, htmlBody := inviteEmail(heading, message, "Accept Invite", link)

	// Send in the background, but wait up to 15 seconds to tell the admin how it went.
	done := make(chan error, 1)
	go func() { done <- sendEmail(email, subject, text, htmlBody) }()
	var sendErr error
	select {
	case sendErr = <-done:
	case <-time.After(15 * time.Second):
		sendErr = errors.New("the email server took too long to answer")
	}

	data := PageData{User: u}
	switch {
	case sendErr == nil:
		log.Println("Invite email handed to the mail server for", email)
		data.Info = "Invitation handed to the mail server for " + email + " ✅ It usually arrives within a minute. " +
			"If " + email + " can't find it, ask them to check Spam, or share the link below with them directly."
		data.InviteLink = link
	case errors.Is(sendErr, errEmailNotConfigured):
		log.Println("Invite link for", email, ":", link)
		data.Info = "Email isn't set up yet, so nothing was emailed. Open Admin > Email Settings, add your mail account, and every invite will send by itself. For now, send this link to " + email + " yourself:"
		data.InviteLink = link
	default:
		log.Println("Could not email invite to", email, ":", sendErr)
		hint := ""
		if msg := sendErr.Error(); strings.Contains(msg, "535") || strings.Contains(msg, "Username and Password not accepted") {
			hint = " The mail server refused the login: check the address and app password on Admin > Email Settings. Gmail needs an App Password, and Brevo needs its SMTP key (never your normal password)."
		}
		data.Error = "The invitation was saved, but the email could not be sent (" + sendErr.Error() + ")." + hint + " You can send this link yourself:"
		data.InviteLink = link
	}
	render(w, page, data)
}

// ---------- Email diary page + test email ----------

func emailsPageHandler(w http.ResponseWriter, r *http.Request) {
	u := requireAdmin(w, r)
	if u == nil {
		return
	}
	data := PageData{User: u, Emails: recentEmails(100)}
	switch r.URL.Query().Get("test") {
	case "sent":
		data.Info = "Test email sent to " + u.Email + " ✅ Open your inbox (and check Spam) to see it."
	case "failed":
		data.Error = "The test email could not be sent. Look at the newest line in the table below to see why."
	case "off":
		data.Error = "Email is not set up yet. Open Admin > Email Settings and add your mail account first."
	}
	render(w, "admin-emails.html", data)
}

// Sends a test email to the admin's OWN address, so you can check everything works.
func emailTestHandler(w http.ResponseWriter, r *http.Request) {
	u := requireAdmin(w, r)
	if u == nil {
		return
	}
	heading := "Test email ✅"
	message := "Hi " + u.FullName + "! If you can read this, PathFinder can send emails. " +
		"This is how the invitation emails will look to the people you invite."
	text, htmlBody := inviteEmail(heading, message, "Open the admin page", baseURLFor(r)+"/admin")

	err := sendEmail(u.Email, "PathFinder test email ✅", text, htmlBody)
	result := "sent"
	switch {
	case errors.Is(err, errEmailNotConfigured):
		result = "off"
	case err != nil:
		result = "failed"
	}
	http.Redirect(w, r, "/admin/emails?test="+result, http.StatusSeeOther)
}

// A row in the "Check Users" table. Passwords are never included.
type UserRow struct {
	FullName string
	Username string
	Email    string
	Role     string
	Online   bool
}

func usersPageHandler(w http.ResponseWriter, r *http.Request) {
	u := requireAdmin(w, r)
	if u == nil {
		return
	}

	all := listUsers()
	rows := make([]UserRow, 0, len(all))
	online := 0
	for _, x := range all {
		on := isOnline(x.ID)
		if on {
			online++
		}
		rows = append(rows, UserRow{
			FullName: x.FullName,
			Username: x.Username,
			Email:    x.Email,
			Role:     x.RoleLabel(),
			Online:   on,
		})
	}
	render(w, "admin-users.html", PageData{User: u, UserRows: rows, TotalUsers: len(rows), OnlineUsers: online})
}

// ---------- Accepting an invitation (opened from the email link) ----------

func acceptPageData(r *http.Request, token string, forUser bool) (PageData, bool) {
	inv, ok := findInvite(token)
	if !ok || inv.ForUser != forUser { // an admin link can't be used on the user page, and vice versa
		return PageData{Error: "This invitation link is not valid. Please ask the person who invited you to send it again."}, false
	}
	data := PageData{Token: token, InviteEmail: inv.Email}
	if existing, found := findUserByEmail(inv.Email); found {
		data.Existing = true
		data.AlreadyMember = existing.IsAdmin()
		if me := currentUser(r); me != nil && me.ID == existing.ID {
			data.CanAccept = true
		}
	}
	return data, true
}

func acceptAdminPageHandler(w http.ResponseWriter, r *http.Request) {
	data, ok := acceptPageData(r, r.URL.Query().Get("token"), false)
	data.User = currentUser(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
	}
	render(w, "accept-admin.html", data)
}

func acceptAdminHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	token := r.FormValue("token")

	data, ok := acceptPageData(r, token, false)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		render(w, "accept-admin.html", data)
		return
	}
	fail := func(msg string) {
		data.Error = msg
		data.FullName = strings.TrimSpace(r.FormValue("full_name"))
		data.Username = strings.TrimSpace(r.FormValue("username"))
		w.WriteHeader(http.StatusBadRequest)
		render(w, "accept-admin.html", data)
	}

	// Case 1: they already have an account and are logged into it -> upgrade it.
	if data.Existing {
		if !data.CanAccept {
			fail("An account with this email already exists. Please log in with it first, then open the invitation link again.")
			return
		}
		me := currentUser(r)
		if !me.IsAdmin() {
			if err := setUserRole(me.ID, RoleAdmin); err != nil {
				log.Println("Could not promote user:", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	// Case 2: brand-new person -> create their admin account.
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if fullName == "" || username == "" || password == "" {
		fail("All fields are required")
		return
	}
	if password != r.FormValue("confirm_password") {
		fail("The two passwords do not match - please type them again")
		return
	}
	if !isValidPassword(password) {
		fail("Password must contain both letters and numbers")
		return
	}
	if len(password) > 72 {
		fail("Password is too long (max 72 characters)")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	newUser, err := createUser(fullName, username, data.InviteEmail, string(hash), RoleAdmin)
	switch {
	case errors.Is(err, ErrUsernameTaken):
		fail("That username is already taken")
		return
	case errors.Is(err, ErrEmailTaken):
		fail("An account with this email already exists")
		return
	case err != nil:
		log.Println("Error saving admin:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := startSession(w, newUser.ID); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// ---------- A normal user accepting an invitation (opened from the email link) ----------

func joinPageHandler(w http.ResponseWriter, r *http.Request) {
	data, ok := acceptPageData(r, r.URL.Query().Get("token"), true)
	data.User = currentUser(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
	}
	render(w, "join.html", data)
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	token := r.FormValue("token")

	data, ok := acceptPageData(r, token, true)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		render(w, "join.html", data)
		return
	}
	fail := func(msg string) {
		data.Error = msg
		data.FullName = strings.TrimSpace(r.FormValue("full_name"))
		data.Username = strings.TrimSpace(r.FormValue("username"))
		w.WriteHeader(http.StatusBadRequest)
		render(w, "join.html", data)
	}

	if data.Existing {
		fail("An account with this email already exists. Please log in instead.")
		return
	}

	fullName := strings.TrimSpace(r.FormValue("full_name"))
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if fullName == "" || username == "" || password == "" {
		fail("All fields are required")
		return
	}
	if password != r.FormValue("confirm_password") {
		fail("The two passwords do not match - please type them again")
		return
	}
	if !isValidPassword(password) {
		fail("Password must contain both letters and numbers")
		return
	}
	if len(password) > 72 {
		fail("Password is too long (max 72 characters)")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	newUser, err := createUser(fullName, username, data.InviteEmail, string(hash), "") // "" = normal user
	switch {
	case errors.Is(err, ErrUsernameTaken):
		fail("That username is already taken")
		return
	case errors.Is(err, ErrEmailTaken):
		fail("An account with this email already exists")
		return
	case err != nil:
		log.Println("Error saving invited user:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	go sendWelcomeEmail(data.InviteEmail, fullName)
	if err := startSession(w, newUser.ID); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

// ---------- Logout goodbye ----------

func goodbyePageHandler(w http.ResponseWriter, r *http.Request) {
	render(w, "goodbye.html", PageData{})
}
