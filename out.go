package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
)

// ---------- The way out ----------
//
// Search results do not link straight to the outside world. They point at /out,
// which writes two lines in history (what was searched, and where the visitor
// went), then redirects. That is how the History page can say
// "Searched “naruto” in Entertainment → went to JustWatch".

// outHandler records a search result click and sends the visitor on.
func outHandler(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		guestRedirect(w, r)
		return
	}

	q := r.URL.Query()
	dest := strings.TrimSpace(q.Get("d"))
	name := strings.TrimSpace(q.Get("n"))
	query := strings.TrimSpace(q.Get("q"))
	section := strings.TrimSpace(q.Get("s"))

	// Only ever redirect to a normal web address, never to something odd.
	parsed, err := url.Parse(dest)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		http.Error(w, "That link is not a web address", http.StatusBadRequest)
		return
	}

	if err := addResultEvent(u.ID, section, query, name, dest); err != nil {
		log.Println("Could not record the search click:", err)
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}
