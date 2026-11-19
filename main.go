package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

// One struct that every page can use. Each page only fills what it needs.
type PageData struct {
	User   *User // nil = guest
	Recent []HistoryEntry

	// Browse / fraction / topic pages
	Fractions []Fraction
	Fraction  Fraction
	Topic     Topic

	// The Programming room: its three tabs
	Tab          string     // "docs" or "documentaries" ("" = the topics tab)
	Languages    []Topic    // the languages to choose from
	Current      Topic      // the chosen language
	Links        []Resource // docs or documentary links for it
	Locked       bool       // topics not clickable yet
	DocsUnlocked bool       // the visitor has read a language's docs
	Theme        string     // css theme key for <body data-theme="...">
	Count        int        // how many places a topic has

	// Admin pages
	Emails        []EmailRecord
	HeadAdmin     *User
	UserRows      []UserRow
	TotalUsers    int
	OnlineUsers   int
	Info          string // green message
	InviteLink    string // link to share by hand if the email could not be sent
	Token         string // invite token (accept page)
	InviteEmail   string
	Existing      bool // the invited email already has an account
	CanAccept     bool // ...and the visitor is logged into it
	AlreadyMember bool // ...and that account is already an admin

	Error     string
	Created   bool // "account created" message on the login page
	NeedLogin bool // show the "please sign up" popup on page load

	// Keeps what the user typed when a form comes back with an error
	FullName string
	Username string
	Email    string
}

// render builds the page in memory first, so a template error
// never sends half a page to the browser.
func render(w http.ResponseWriter, name string, data PageData) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Println(err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// ---------- Public pages ----------

func landingPageHandler(w http.ResponseWriter, r *http.Request) {
	render(w, "landing.html", PageData{Theme: "base"})
}

// The Browse page: every fraction as a big box.
// Anyone can SEE it. Guests get the popup when they click one.
func browseHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	render(w, "browse.html", PageData{
		User:      user,
		Fractions: fractions,
		Theme:     "base",
		NeedLogin: user == nil && r.URL.Query().Get("login_required") == "1",
	})
}

// ---------- Logged-in pages ----------

// guestRedirect sends a guest back to the browse page with the popup turned on.
func guestRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/browse?login_required=1", http.StatusSeeOther)
}

// The page people land on after logging in: the four explanation boxes,
// the Browse button and the fractions.
func homePageHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	render(w, "home.html", PageData{
		User:      user,
		Fractions: fractions,
		Recent:    recentHistory(user.ID, 5),
		Theme:     "base",
	})
}

// One fraction: its own colour, its own topics (and the programming fork).
func fractionHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		guestRedirect(w, r)
		return
	}

	f, ok := fractionBySlug[r.PathValue("slug")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	render(w, "fraction.html", PageData{User: user, Fraction: f, Theme: f.Theme})
}

// One topic: the 15 places you can be redirected to.
func topicHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		guestRedirect(w, r)
		return
	}

	slug := r.PathValue("slug")
	t, ok := topicBySlug[slug]
	if !ok {
		http.NotFound(w, r)
		return
	}

	data := PageData{User: user, Topic: t, Count: len(t.Resources), Theme: "base"}
	if fSlug, ok := fractionOfTopic[slug]; ok {
		data.Theme = themeFor(fSlug)
		if f, ok := fractionBySlug[fSlug]; ok {
			data.Fraction = f
		}
	}
	render(w, "topic.html", data)
}

// Saves the visit in the user's history, then sends them to the site.
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		guestRedirect(w, r)
		return
	}

	slug := r.PathValue("slug")
	resourceID := r.PathValue("resourceID")

	t, ok := topicBySlug[slug]
	if !ok {
		http.NotFound(w, r)
		return
	}

	for _, resource := range t.Resources {
		if resource.ID == resourceID {
			if err := addHistory(user.ID, slug, resource.ID, resource.Name); err != nil {
				log.Println("Could not save history:", err) // don't block the redirect
			}
			log.Printf("%s -> %s (%s)", user.Username, resource.Name, resource.URL)
			http.Redirect(w, r, resource.URL, http.StatusFound)
			return
		}
	}

	http.NotFound(w, r)
}

func main() {
	loadDotEnv() // read settings from the .env file first
	logEmailStatus()
	initDB()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", landingPageHandler) // exactly "/", anything else = 404
	mux.HandleFunc("GET /browse", browseHandler)
	mux.HandleFunc("GET /home", homePageHandler)
	mux.HandleFunc("GET /fraction/{slug}", fractionHandler)
	mux.HandleFunc("GET /fraction/programming/{tab}", programmingTabHandler)
	mux.HandleFunc("GET /fraction/programming/{tab}/{lang}", programmingTabHandler)
	mux.HandleFunc("GET /topic/{slug}", topicHandler)
	mux.HandleFunc("GET /go/{slug}/{resourceID}", redirectHandler)

	// Old links keep working.
	mux.HandleFunc("GET /languages", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/browse", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /language/{slug}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/topic/"+r.PathValue("slug"), http.StatusSeeOther)
	})

	mux.HandleFunc("GET /signup", signupPageHandler)
	mux.HandleFunc("POST /signup", signupHandler)
	mux.HandleFunc("GET /login", loginPageHandler)
	mux.HandleFunc("POST /login", loginHandler)
	mux.HandleFunc("POST /logout", logoutHandler)
	mux.HandleFunc("GET /goodbye", goodbyePageHandler)
	mux.HandleFunc("GET /admin/login", adminLoginPageHandler)
	mux.HandleFunc("POST /admin/login", adminLoginHandler)

	mux.HandleFunc("GET /admin", adminPageHandler)
	mux.HandleFunc("GET /admin/invite", inviteAdminPageHandler)
	mux.HandleFunc("POST /admin/invite", inviteAdminHandler)
	mux.HandleFunc("GET /admin/invite-user", inviteUserPageHandler)
	mux.HandleFunc("POST /admin/invite-user", inviteUserHandler)
	mux.HandleFunc("GET /admin/users", usersPageHandler)
	mux.HandleFunc("GET /admin/emails", emailsPageHandler)
	mux.HandleFunc("POST /admin/emails/test", emailTestHandler)
	mux.HandleFunc("GET /join", joinPageHandler)
	mux.HandleFunc("POST /join", joinHandler)
	mux.HandleFunc("GET /admin/accept", acceptAdminPageHandler)
	mux.HandleFunc("POST /admin/accept", acceptAdminHandler)

	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
