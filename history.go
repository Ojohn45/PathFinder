package main

import (
	"net/http"
	"net/url"
	"time"
)

// ---------- History ----------
//
// Every time a visitor opens one of the fifteen places, the visit is written to
// storage/history.json. This page shows the whole trail, newest first, grouped
// by day, with a link to go straight back to that place.

// HistoryRow is one line on the page: the visit plus the names it belongs to.
type HistoryRow struct {
	Kind         string // "", "search" or "result"
	ResourceName string
	Query        string
	DestName     string
	DestURL      string
	LanguageName string // the topic: a language, "Anime", "Read Manga"...
	SectionName  string
	SectionSlug  string
	Link         string // /go/<topic>/<resource> for a visit, the outside URL for a result
	Icon         string // the domain whose favicon is shown next to the row
	When         time.Time
}

// hostOf pulls the domain out of a URL, for the favicon service.
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Hostname()
}

// HistoryDay groups the visits of one calendar day.
type HistoryDay struct {
	Heading string
	Rows    []HistoryRow
}

// allHistory returns every visit for one user, newest first.
func allHistory(userID int) []HistoryEntry {
	mu.Lock()
	defer mu.Unlock()
	var out []HistoryEntry
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].UserID == userID {
			out = append(out, history[i])
		}
	}
	return out
}

// historyRows turns raw visits into rows with human names attached.
func historyRows(entries []HistoryEntry) []HistoryRow {
	rows := make([]HistoryRow, 0, len(entries))
	for _, e := range entries {
		row := HistoryRow{
			Kind:         e.Kind,
			ResourceName: e.ResourceName,
			Query:        e.Query,
			DestName:     e.DestName,
			DestURL:      e.DestURL,
			When:         e.ClickedAt,
			Link:         "/go/" + e.LanguageSlug + "/" + e.ResourceID,
			LanguageName: e.LanguageSlug,
			SectionName:  "PathFinder",
			SectionSlug:  e.SectionSlug,
		}

		// A search or a search result belongs to a section, not to a topic.
		if e.Kind == "search" || e.Kind == "result" {
			if sec, ok := sectionBySlug[e.SectionSlug]; ok {
				row.SectionName = sec.Name
			}
			if e.Kind == "result" {
				row.Link = e.DestURL
				row.Icon = hostOf(e.DestURL)
			}
			rows = append(rows, row)
			continue
		}

		if t, ok := topicBySlug[e.LanguageSlug]; ok {
			row.LanguageName = t.Name
			for _, res := range t.Resources {
				if res.ID == e.ResourceID {
					row.Icon = hostOf(res.URL)
					break
				}
			}
		}
		if slug, ok := sectionOfTopic[e.LanguageSlug]; ok {
			if sec, ok := sectionBySlug[slug]; ok {
				row.SectionName = sec.Name
				row.SectionSlug = sec.Slug
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// groupByDay puts rows under Today / Yesterday / a date heading.
func groupByDay(rows []HistoryRow) []HistoryDay {
	var days []HistoryDay
	now := time.Now()
	for _, row := range rows {
		when := row.When.Local()
		heading := when.Format("Mon 2 Jan 2006")
		switch {
		case sameDay(when, now):
			heading = "Today"
		case sameDay(when, now.AddDate(0, 0, -1)):
			heading = "Yesterday"
		}
		if len(days) == 0 || days[len(days)-1].Heading != heading {
			days = append(days, HistoryDay{Heading: heading})
		}
		days[len(days)-1].Rows = append(days[len(days)-1].Rows, row)
	}
	return days
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// historyPageHandler serves /history.
// withinRange says whether a time falls inside the chosen window.
// Weeks start on Monday.
func withinRange(t time.Time, span string) bool {
	now := time.Now()
	t, now = t.Local(), now.Local()
	switch span {
	case "today":
		return sameDay(t, now)
	case "week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(weekday - 1))
		return !t.Before(start)
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return !t.Before(start)
	case "year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return !t.Before(start)
	}
	return true // "all"
}

func historyPageHandler(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		guestRedirect(w, r)
		return
	}
	span := r.URL.Query().Get("range")
	switch span {
	case "today", "week", "month", "year":
	default:
		span = "all"
	}

	entries := allHistory(u.ID)
	if span != "all" {
		kept := make([]HistoryEntry, 0, len(entries))
		for _, e := range entries {
			if withinRange(e.ClickedAt, span) {
				kept = append(kept, e)
			}
		}
		entries = kept
	}

	rows := historyRows(entries)
	data := PageData{
		User:      u,
		Theme:     "base",
		TotalSeen: len(entries),
		History:   groupByDay(rows),
		Nav:       "history",
		Range:     span,
	}
	render(w, "history.html", data)
}
