package main

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// ---------- Account settings ----------
//
// A logged-in user can change three things: their full name, their username and
// their password. The email address and the phone number are deliberately NOT
// editable here: they are how the account is identified and how we reach them,
// so changing them needs a verified flow, not a text box.

func settingsPageHandler(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		guestRedirect(w, r)
		return
	}
	data := PageData{User: u, Theme: "base", Nav: "settings"}
	data.FullName = u.FullName
	data.Username = u.Username
	data.Email = u.Email
	switch r.URL.Query().Get("ok") {
	case "name":
		data.Info = "Your name has been updated."
	case "username":
		data.Info = "Your username has been updated. Use it next time you log in."
	case "password":
		data.Info = "Your password has been changed."
	}
	render(w, "settings.html", data)
}

// updateProfile saves the name and username, refusing a username somebody else has.
func updateProfile(userID int, fullName, username string) error {
	fullName = strings.TrimSpace(fullName)
	username = strings.TrimSpace(username)
	if fullName == "" || username == "" {
		return errors.New("your name and username cannot be empty")
	}
	if len(username) < 3 {
		return errors.New("the username needs at least 3 characters")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, u := range users {
		if u.ID != userID && strings.EqualFold(u.Username, username) {
			return errors.New("that username is already taken")
		}
	}
	for i := range users {
		if users[i].ID == userID {
			oldName, oldUser := users[i].FullName, users[i].Username
			users[i].FullName = fullName
			users[i].Username = username
			if err := saveJSON(usersFile, users); err != nil {
				users[i].FullName, users[i].Username = oldName, oldUser
				return err
			}
			return nil
		}
	}
	return errors.New("no such user")
}

func settingsNameHandler(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		guestRedirect(w, r)
		return
	}
	fullName := r.FormValue("full_name")
	username := r.FormValue("username")

	if err := updateProfile(u.ID, fullName, username); err != nil {
		fresh := currentUser(r)
		data := PageData{User: fresh, Theme: "base", Error: err.Error(), Nav: "settings"}
		if fresh != nil {
			data.FullName, data.Username, data.Email = fullName, username, fresh.Email
		}
		w.WriteHeader(http.StatusBadRequest)
		render(w, "settings.html", data)
		return
	}

	changedUsername := !strings.EqualFold(strings.TrimSpace(username), u.Username)
	if changedUsername {
		http.Redirect(w, r, "/settings?ok=username", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/settings?ok=name", http.StatusSeeOther)
}

func settingsPasswordHandler(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		guestRedirect(w, r)
		return
	}
	current := r.FormValue("current_password")
	next := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	fail := func(msg string) {
		data := PageData{User: u, Theme: "base", Error: msg, FullName: u.FullName, Username: u.Username, Email: u.Email, Nav: "settings"}
		w.WriteHeader(http.StatusBadRequest)
		render(w, "settings.html", data)
	}

	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(current)) != nil {
		fail("That is not your current password")
		return
	}
	if next != confirm {
		fail("The two new passwords do not match")
		return
	}
	if !isValidPassword(next) {
		fail("The new password needs both letters and numbers")
		return
	}
	if len(next) > 72 {
		fail("That password is too long (max 72 characters)")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := setUserPassword(u.ID, string(hash)); err != nil {
		log.Println("Could not change the password:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings?ok=password", http.StatusSeeOther)
}
