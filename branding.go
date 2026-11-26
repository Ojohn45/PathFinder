package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ---------- The tab icon ----------
//
// Browsers show a small picture next to the page title. PathFinder ships five
// drawings of its own and the head admin picks one on Admin > Branding. The
// choice lives in storage/settings.json next to the mail settings, and the
// chosen file is served at /favicon.ico so every page points at one address.

// IconOption is one choice in the picker.
type IconOption struct {
	ID    string // file name stem, e.g. "option-2-path"
	Name  string
	Blurb string
}

var iconOptions = []IconOption{
	{"option-1-p", "The P", "A bold P on the blue wash. The simplest mark, easiest to read at 16px."},
	{"option-2-path", "The path", "The route line with an arrow: the same idea as the wordmark."},
	{"option-3-climb", "The climb", "A path going up and to the right, ending in an arrowhead."},
	{"option-4-fork", "The fork", "One way in, two ways on. Reads as choosing a section."},
	{"option-5-pin", "The pin", "A path that ends at a marker, like arriving somewhere."},
}

const defaultIcon = "option-2-path"

// currentIcon returns the chosen icon, falling back to the default when the
// saved value is missing or no longer one of the options.
func currentIcon() string {
	mailMu.RLock()
	choice := mailSet.Favicon
	mailMu.RUnlock()

	for _, o := range iconOptions {
		if o.ID == choice {
			return choice
		}
	}
	return defaultIcon
}

// saveIconChoice writes just the icon field, leaving the mail settings alone.
func saveIconChoice(choice string) error {
	valid := false
	for _, o := range iconOptions {
		if o.ID == choice {
			valid = true
			break
		}
	}
	if !valid {
		return os.ErrInvalid
	}

	mailMu.Lock()
	defer mailMu.Unlock()
	next := mailSet
	next.Favicon = choice

	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err := storeWriteSettings(data); err != nil {
		return err
	}

	mailSet = next
	return nil
}

// serveIcon answers /favicon.ico and /apple-touch-icon.png with the chosen file.
func serveIcon(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := currentIcon()
		file := "static/icons/" + name + ".ico"
		if kind == "touch" {
			file = "static/icons/" + name + "-180.png"
		}
		w.Header().Set("Cache-Control", "public, max-age=300")
		http.ServeFile(w, r, filepath.Clean(file))
	}
}

// brandingPageHandler shows the five drawings with the current one marked.
func brandingPageHandler(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil || !u.IsAdmin() {
		guestRedirect(w, r)
		return
	}
	data := PageData{User: u, Theme: "base", Nav: "admin", NoBack: true}
	data.Icons = iconOptions
	data.IconChoice = currentIcon()
	data.Info = r.URL.Query().Get("saved")
	render(w, "admin-branding.html", data)
}

// brandingSaveHandler stores the pick and sends the admin back to the page.
func brandingSaveHandler(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil || !u.IsAdmin() {
		guestRedirect(w, r)
		return
	}
	choice := strings.TrimSpace(r.FormValue("icon"))
	if err := saveIconChoice(choice); err != nil {
		data := PageData{User: u, Theme: "base", Nav: "admin", NoBack: true}
		data.Icons = iconOptions
		data.IconChoice = currentIcon()
		data.Error = "That choice was not one of the drawings. Pick one from the list."
		w.WriteHeader(http.StatusBadRequest)
		render(w, "admin-branding.html", data)
		return
	}
	http.Redirect(w, r, "/admin/branding?saved=1", http.StatusSeeOther)
}
