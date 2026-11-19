package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ---------- The Programming room ----------
//
// Programming is the one fraction with three tabs:
//
//	/fraction/programming                      the topics (locked until you read a doc)
//	/fraction/programming/docs                 pick a language, read its documentation
//	/fraction/programming/docs/{lang}          that language's docs
//	/fraction/programming/documentaries        pick a language first...
//	/fraction/programming/documentaries/{lang} ...then see where to watch
//
// The topics stay unclickable until the visitor has opened a language's docs,
// which is the point: read first, then go out to the sites.

// learningFor is the straight-to-learning route: the same fifteen curated
// places a topic holds, but linked through /go so the visit is saved.
func learningFor(t Topic) []Resource {
	out := make([]Resource, 0, len(t.Resources))
	for _, r := range t.Resources {
		r.URL = "/go/" + t.Slug + "/" + r.ID
		out = append(out, r)
	}
	return out
}

// docsFor returns the reading material for one language: its documentation,
// references and manuals, pulled from the same curated list the topics use.
func docsFor(t Topic) []Resource {
	keys := []string{"doc", "manual", "reference", "guide", "spec", "handbook", "faq"}
	var out []Resource
	for _, r := range t.Resources {
		hay := strings.ToLower(r.Name + " " + r.Description)
		for _, k := range keys {
			if strings.Contains(hay, k) {
				out = append(out, r)
				break
			}
		}
	}
	if len(out) < 4 { // some languages name their docs differently
		out = out[:0]
		for i, r := range t.Resources {
			if i >= 6 {
				break
			}
			out = append(out, r)
		}
	}
	out = append(out, Resource{
		ID:          "devdocs",
		Name:        "DevDocs",
		URL:         "https://devdocs.io",
		Description: "Every language's documentation in one fast reader",
	})
	return out
}

// documentariesFor answers "where can I actually watch something about this?".
func documentariesFor(t Topic) []Resource {
	q := url.QueryEscape(t.Name + " programming documentary")
	out := []Resource{
		{ID: "doc-yt", Name: "YouTube · " + t.Name + " documentaries",
			URL:         "https://www.youtube.com/results?search_query=" + q,
			Description: "Full films, conference talks and channel deep-dives"},
		{ID: "doc-ia", Name: "Internet Archive · " + t.Name,
			URL:         "https://archive.org/search?query=" + url.QueryEscape(t.Name+" documentary"),
			Description: "Free archive films, lectures and old television"},
	}
	classics := []struct{ name, blurb string }{
		{"Revolution OS", "The birth of open source, told by the people who did it"},
		{"General Magic", "The phone that arrived twenty years too early"},
		{"AlphaGo", "The match that changed what machines can do"},
		{"The Internet's Own Boy", "Aaron Swartz and the fight over the open web"},
		{"Silicon Cowboys", "Compaq against IBM, the original underdog story"},
	}
	for i, c := range classics {
		out = append(out, Resource{
			ID:          fmt.Sprintf("doc-classic-%d", i),
			Name:        c.name,
			URL:         "https://www.justwatch.com/us/search?q=" + url.QueryEscape(c.name),
			Description: c.blurb + " · see where to watch it",
		})
	}
	return out
}

// programmingTabHandler serves the Docs and Watch documentaries tabs.
func programmingTabHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		guestRedirect(w, r)
		return
	}

	tab := r.PathValue("tab")

	// The page a topic click lands on when JavaScript is off.
	if tab == "notice" {
		render(w, "prog-notice.html", PageData{
			User: user, Theme: "prog",
			Fraction: fractionBySlug["programming"],
		})
		return
	}

	if tab != "docs" && tab != "documentaries" && tab != "learning" {
		http.NotFound(w, r)
		return
	}

	data := PageData{
		User:      user,
		Theme:     "prog",
		Fraction:  fractionBySlug["programming"],
		Tab:       tab,
		Languages: programmingTopicList(),
	}

	lang := r.PathValue("lang")
	if lang == "" {
		render(w, "prog-tab.html", data)
		return
	}

	t, ok := programmingTopics[lang]
	if !ok {
		http.NotFound(w, r)
		return
	}
	data.Current = t

	switch tab {
	case "docs":
		data.Links = docsFor(t)
	case "documentaries":
		data.Links = documentariesFor(t)
	case "learning":
		data.Links = learningFor(t)
	}
	render(w, "prog-tab.html", data)
}
