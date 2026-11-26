package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// faviconHost is a template helper: it turns a link into the domain whose logo
// should sit next to it. A blank result simply hides the picture.
func faviconHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Hostname()
}

var templates = template.Must(template.New("").Funcs(template.FuncMap{
	"favicon":  faviconHost,
	"initial":  initialLetter,
	"iconName": func() string { return currentIcon() },
	"faviconOf": func(slug, id string) string {
		if t, ok := topicBySlug[slug]; ok {
			for _, r := range t.Resources {
				if r.ID == id {
					return faviconHost(r.URL)
				}
			}
		}
		return ""
	},
}).ParseGlob("templates/*.html"))

// One struct that every page can use. Each page only fills what it needs.
type PageData struct {
	User   *User // nil = guest
	Recent []HistoryEntry

	// Browse / section / topic pages
	Sections []Section
	Section  Section
	Topic    Topic

	// The Programming room: its three doors
	Tab       string     // "learning", "docs" or "documentaries" ("" = the topics tab)
	Languages []Topic    // the languages to choose from
	Current   Topic      // the chosen language
	Links     []Resource // the places for it
	Theme     string     // css theme key for <html data-theme="...">
	Count     int        // how many places a topic has

	// Places shown straight on a section page (Read Manga, Read Manhwa)
	Sites []Resource

	// SitesLabel is the heading above the places list, e.g. "Where to read"
	// for comics or "Where to see clothing & accessories" for clothing.
	SitesLabel string

	// Set on a home base (the home page, the admin dashboard): no back button
	NoBack bool

	// Set on the three home bases (landing, home, admin): the logo sits in the
	// middle of the sidebar instead of on the left.
	CenterLogo bool

	// Nav marks which sidebar link is the current page.
	Nav string

	// History page: "today", "week", "month", "year" or "all".
	Range string

	// Branding page: the tab icon choices and which one is live.
	Icons      []IconOption
	IconChoice string

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

	// Searching inside a section, and the visit history
	Search    SectionSearch
	Query     string
	History   []HistoryDay
	TotalSeen int

	// Email Settings page (head admin)
	Mail        MailSettings
	MailSaved   bool // something was typed on the page and saved
	MailFromEnv bool // the server sets SMTP_USER in its environment
	MailReady   bool // enough to attempt a send

	Error     string
	Created   bool // "account created" message on the login page
	NeedLogin bool // show the "please sign up" popup on page load

	// Keeps what the user typed when a form comes back with an error
	FullName string
	Username string
	Email    string
	Phone    string
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
	render(w, "landing.html", PageData{Theme: "base", CenterLogo: true})
}

// The Browse page: every section as a big box.
// Anyone can SEE it. Guests get the popup when they click one.
func browseHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	render(w, "browse.html", PageData{
		User:      user,
		Sections:  sections,
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
// the Browse button and the sections.
func homePageHandler(w http.ResponseWriter, r *http.Request) {
	// The home page is also the landing page, so guests are welcome here.
	// A guest simply sees the two "create account" buttons instead.
	user := currentUser(r)
	var recent []HistoryEntry
	if user != nil {
		recent = recentHistory(user.ID, 5)
	}
	render(w, "home.html", PageData{
		User:       user,
		Sections:   sections,
		Recent:     recent,
		Theme:      "base",
		NoBack:     true,
		CenterLogo: true,
		Nav:        "home",
	})
}

// One section: its own colour, its own topics (and the programming fork).
func sectionHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		guestRedirect(w, r)
		return
	}

	f, ok := sectionBySlug[r.PathValue("slug")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	data := PageData{User: user, Section: f, Theme: f.Theme}
	if f.Direct && len(f.Topics) > 0 {
		data.Sites = learningFor(f.Topics[0])
		data.SitesLabel = sitesLabelFor(f)
	}
	render(w, "section.html", data)
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
	if fSlug, ok := sectionOfTopic[slug]; ok {
		data.Theme = themeFor(fSlug)
		if f, ok := sectionBySlug[fSlug]; ok {
			data.Section = f
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
	loadDotEnv()       // optional: settings from a .env file
	loadMailSettings() // the mail account saved on Admin > Email Settings
	logEmailStatus()
	initDB()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", landingPageHandler) // exactly "/", anything else = 404
	mux.HandleFunc("GET /browse", browseHandler)
	mux.HandleFunc("GET /home", homePageHandler)
	mux.HandleFunc("GET /section/{slug}", sectionHandler)
	for _, slug := range []string{"programming", "entertainment", "manga", "manhwa", "comix"} {
		mux.HandleFunc("GET /section/"+slug+"/search", searchHandlerFor(slug))
	}
	mux.HandleFunc("GET /history", historyPageHandler)
	mux.HandleFunc("GET /out", outHandler)
	// the tab icon: one address, whichever drawing is chosen
	mux.HandleFunc("GET /favicon.ico", serveIcon("ico"))
	mux.HandleFunc("GET /apple-touch-icon.png", serveIcon("touch"))
	mux.HandleFunc("GET /admin/branding", brandingPageHandler)
	mux.HandleFunc("POST /admin/branding", brandingSaveHandler)
	mux.HandleFunc("GET /settings", settingsPageHandler)
	mux.HandleFunc("POST /settings/name", settingsNameHandler)
	mux.HandleFunc("POST /settings/password", settingsPasswordHandler)
	mux.HandleFunc("GET /section/programming/{tab}", programmingTabHandler)
	mux.HandleFunc("GET /section/programming/{tab}/{lang}", programmingTabHandler)
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
	// There is no separate admin door any more: the head admin signs in on the
	// normal login page with their registered number. Old links still land there.
	mux.HandleFunc("GET /admin/login", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /admin/login", adminLoginHandler)

	mux.HandleFunc("GET /admin", adminPageHandler)
	mux.HandleFunc("GET /admin/invite", inviteAdminPageHandler)
	mux.HandleFunc("POST /admin/invite", inviteAdminHandler)
	mux.HandleFunc("GET /admin/invite-user", inviteUserPageHandler)
	mux.HandleFunc("POST /admin/invite-user", inviteUserHandler)
	mux.HandleFunc("GET /admin/users", usersPageHandler)
	mux.HandleFunc("GET /admin/emails", emailsPageHandler)
	mux.HandleFunc("GET /admin/email-settings", emailSettingsPageHandler)
	mux.HandleFunc("POST /admin/email-settings", emailSettingsSaveHandler)
	mux.HandleFunc("POST /admin/emails/test", emailTestHandler)
	mux.HandleFunc("GET /join", joinPageHandler)
	mux.HandleFunc("POST /join", joinHandler)
	mux.HandleFunc("GET /admin/accept", acceptAdminPageHandler)
	mux.HandleFunc("POST /admin/accept", acceptAdminHandler)

	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Hosts like Render, Railway and Fly hand you a port in $PORT; locally we
	// default to 8080 so `go run .` keeps working the way you expect.
	addr := ":" + getenv("PORT", "8080")
	log.Println("Server starting on http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// initialLetter returns the first character of a name, for the fallback logo
// tile shown when a site's own icon cannot be loaded.
func initialLetter(name string) string {
	for _, r := range name {
		if r == ' ' {
			continue
		}
		return strings.ToUpper(string(r))
	}
	return "?"
}
